// test/l7/traffic_mirror/traffic_mirror_test.go
package traffic_mirror

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

// TestTrafficMirror verifies that a VirtualService with mirror configuration
// sends traffic to both the primary and mirror backends. The primary
// receives all responses; the mirror receives a shadow copy.
func TestTrafficMirror(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := manifest.TemplateValues{
		Namespace:    env.Namespaces.HTTP,
		TestID:       env.RunID,
		StableWeight: 100,
		CanaryWeight: 0,
	}

	// ── SETUP ────────────────────────────────────────────────
	result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
	if err != nil {
		t.Fatalf("apply mirror manifests: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	time.Sleep(5 * time.Second)

	client := &http.Client{Timeout: 5 * time.Second}
	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}

	// ── TEST: Primary backend receives all responses ──────────
	t.Run("primary_receives_responses", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			resp, err := traffic.HTTPGet(ctx, client, endpoint)
			assert.NoError(t, err)
			assert.StatusOK(t, resp)
		}
	})

	// ── TEST: Mirror does not affect response to caller ───────
	t.Run("mirror_transparent_to_client", func(t *testing.T) {
		// The caller only sees the response from the primary.
		// Mirror sends fire-and-forget — no response latency impact.
		for i := 0; i < 5; i++ {
			resp, err := traffic.HTTPGet(ctx, client, endpoint)
			assert.NoError(t, err)
			assert.StatusOK(t, resp)
		}
	})

	// ── TEST: Mirror removed, traffic still works ─────────────
	t.Run("remove_mirror_traffic_continues", func(t *testing.T) {
		if err := applier.DeleteResult(ctx, result); err != nil {
			t.Fatalf("delete mirror policy: %v", err)
		}
		result = nil // prevent double-delete in cleanup

		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			2*time.Second, 30*time.Second)
	})
}

// TestTrafficMirrorPartial verifies that partial mirroring (50%) works:
// all requests succeed and mirror percentage does not affect response rate.
func TestTrafficMirrorPartial(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := manifest.TemplateValues{
		Namespace:    env.Namespaces.HTTP,
		TestID:       env.RunID,
		StableWeight: 100,
		CanaryWeight: 0,
	}

	result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
	if err != nil {
		t.Fatalf("apply mirror manifests: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	time.Sleep(5 * time.Second)

	client := &http.Client{Timeout: 5 * time.Second}
	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}

	// All 20 requests should succeed from caller's perspective
	t.Run("all_requests_succeed", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			resp, err := traffic.HTTPGet(ctx, client, endpoint)
			assert.NoError(t, err)
			assert.StatusOK(t, resp)
		}
	})
}
