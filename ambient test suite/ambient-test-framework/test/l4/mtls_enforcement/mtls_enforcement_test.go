// test/l4/mtls_enforcement/mtls_enforcement_test.go
package mtls_enforcement

import (
	"context"
	"embed"
	"net/http"
	"testing"
	"time"

	"github.com/yourorg/ambient-test-framework/pkg/assert"
	"github.com/yourorg/ambient-test-framework/pkg/manifest"
	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	pkgwait "github.com/yourorg/ambient-test-framework/pkg/wait"
	"github.com/yourorg/ambient-test-framework/test"
)

//go:embed testdata/*
var testdataFS embed.FS

// TestMTLSEnforcementStrict verifies that STRICT mTLS mode is enforced in
// an ambient-enrolled namespace: mesh-enrolled clients succeed, non-mesh
// plaintext traffic is rejected.
func TestMTLSEnforcementStrict(t *testing.T) {
	test.SkipUnlessLayer(t, "l4")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := manifest.TemplateValues{
		Namespace: env.Namespaces.HTTP,
		TestID:    env.RunID,
	}

	// ── SETUP: Apply PeerAuthentication STRICT ────────────────
	result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
	if err != nil {
		t.Fatalf("apply peer-auth manifests: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	// Wait briefly for the policy to propagate
	time.Sleep(5 * time.Second)

	// ── TEST: Mesh client can reach service ───────────────────
	t.Run("mesh_client_allowed", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		client := &http.Client{Timeout: 5 * time.Second}
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			3*time.Second, 30*time.Second)
	})

	// ── TEST: ztunnel is running and enforcing mTLS ───────────
	t.Run("ztunnel_running", func(t *testing.T) {
		assert.ZTunnelRunning(ctx, t, env.Clientset)
	})

	// ── TEST: Namespace is enrolled in ambient mesh ───────────
	t.Run("namespace_ambient_enrolled", func(t *testing.T) {
		assert.MTLSEnforced(ctx, t, env.Clientset,
			env.Namespaces.HTTP, "podinfo")
	})

	// ── TEST: mTLS mode label is present ─────────────────────
	t.Run("ambient_label_verified", func(t *testing.T) {
		err := pkgwait.UntilNoError(ctx, func(ctx context.Context) error {
			assert.MTLSEnforced(ctx, t, env.Clientset,
				env.Namespaces.HTTP, "podinfo")
			return nil
		}, 2*time.Second, 20*time.Second)
		if err != nil {
			t.Fatalf("ambient label check failed: %v", err)
		}
	})
}

// TestMTLSPeerAuthPermissive verifies that in PERMISSIVE mode both
// mTLS and plaintext connections succeed.
func TestMTLSPeerAuthPermissive(t *testing.T) {
	test.SkipUnlessLayer(t, "l4")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// No PeerAuthentication applied → default is PERMISSIVE in ambient
	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("permissive_allows_plaintext", func(t *testing.T) {
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			3*time.Second, 30*time.Second)
	})
}
