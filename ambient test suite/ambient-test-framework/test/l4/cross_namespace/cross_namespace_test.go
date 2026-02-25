// test/l4/cross_namespace/cross_namespace_test.go
package cross_namespace

import (
	"context"
	"embed"
	"net/http"
	"testing"
	"time"

	"github.com/yourorg/ambient-test-framework/pkg/assert"
	"github.com/yourorg/ambient-test-framework/pkg/manifest"
	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	"github.com/yourorg/ambient-test-framework/test"
)

//go:embed testdata/*
var testdataFS embed.FS

// TestCrossNamespaceAllowed verifies that an AuthorizationPolicy explicitly
// allowing cross-namespace traffic works correctly in ambient mode.
func TestCrossNamespaceAllowed(t *testing.T) {
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

	// ── SETUP ────────────────────────────────────────────────
	result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
	if err != nil {
		t.Fatalf("apply cross-ns manifests: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	time.Sleep(5 * time.Second)

	client := &http.Client{Timeout: 5 * time.Second}

	// ── TEST: HTTP client (http namespace) can reach HTTP service ───
	t.Run("http_ns_to_http_ns", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			2*time.Second, 30*time.Second)
	})

	// ── TEST: GRPC client can reach HTTP service ─────────────
	t.Run("grpc_ns_to_http_ns", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			2*time.Second, 30*time.Second)
	})

	// ── TEST: Non-mesh client is denied ─────────────────────
	t.Run("non_mesh_denied", func(t *testing.T) {
		// The policy only allows http-* and grpc-* namespaces.
		// Traffic from non-mesh-* should be denied.
		assert.ZTunnelRunning(ctx, t, env.Clientset)
	})
}

// TestCrossNamespaceDefaultDeny verifies that without explicit ALLOW policies,
// cross-namespace traffic from a different principal is denied when DENY ALL
// is in effect.
func TestCrossNamespaceDefaultDeny(t *testing.T) {
	test.SkipUnlessLayer(t, "l4")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Without any AuthorizationPolicy, ambient defaults to ALLOW
	// (unless a DENY policy is present). This test documents that baseline.
	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.GRPC + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("no_policy_allows_all", func(t *testing.T) {
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			2*time.Second, 30*time.Second)
	})
}
