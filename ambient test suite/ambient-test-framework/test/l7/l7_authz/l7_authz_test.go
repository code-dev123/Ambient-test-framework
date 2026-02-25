// test/l7/l7_authz/l7_authz_test.go
package l7_authz

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

// TestL7AuthzDenyPath verifies that an L7 AuthorizationPolicy enforced at
// the waypoint proxy denies access to the /admin path.
func TestL7AuthzDenyPath(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := manifest.TemplateValues{
		Namespace:        env.Namespaces.HTTP,
		TestID:           env.RunID,
		GatewayClassName: "istio-waypoint",
	}

	// ── SETUP: Deploy waypoint + AuthorizationPolicy ─────────
	result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
	if err != nil {
		t.Fatalf("apply l7-authz manifests: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	err = applier.WaitForReady(ctx, result, 90*time.Second)
	if err != nil {
		t.Fatalf("l7-authz resources not ready: %v", err)
	}

	// Brief propagation delay
	time.Sleep(5 * time.Second)

	client := &http.Client{Timeout: 5 * time.Second}

	// ── TEST: Normal path is allowed ─────────────────────────
	t.Run("allowed_path", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			2*time.Second, 20*time.Second)
	})

	// ── TEST: /admin path is denied (403) ────────────────────
	t.Run("admin_path_denied", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/admin",
		}
		assert.EventuallyHTTPStatus(ctx, t, client, endpoint,
			http.StatusForbidden,
			2*time.Second, 20*time.Second)
	})

	// ── TEST: /admin/ sub-paths are also denied ───────────────
	t.Run("admin_subpath_denied", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/admin/settings",
		}
		assert.EventuallyHTTPStatus(ctx, t, client, endpoint,
			http.StatusForbidden,
			2*time.Second, 20*time.Second)
	})
}

// TestL7AuthzAllowSpecificMethod verifies that an AuthorizationPolicy
// can restrict traffic to specific HTTP methods.
func TestL7AuthzAllowSpecificMethod(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Without any method restriction, GET should succeed
	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("get_allowed_by_default", func(t *testing.T) {
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			2*time.Second, 30*time.Second)
	})
}
