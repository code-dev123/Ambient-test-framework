// test/l7/header_routing/header_routing_test.go
package header_routing

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

// TestHeaderBasedRouting verifies that an HTTPRoute with header match
// routes requests with x-version: canary to v2 and all other traffic to v1.
func TestHeaderBasedRouting(t *testing.T) {
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

	// ── SETUP: Deploy waypoint and HTTPRoute ─────────────────
	result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
	if err != nil {
		t.Fatalf("apply header-routing manifests: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	err = applier.WaitForReady(ctx, result, 60*time.Second)
	if err != nil {
		t.Fatalf("header-routing resources not ready: %v", err)
	}

	// ── TEST: Request without header goes to default backend ──
	t.Run("no_header_routes_to_default", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/api/info",
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		assert.NoError(t, err)
		assert.StatusOK(t, resp)
	})

	// ── TEST: Request with x-version: canary header is routed ──
	t.Run("canary_header_routes_to_v2", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/api/info",
			Headers: map[string]string{
				"x-version": "canary",
			},
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		assert.NoError(t, err)
		assert.StatusOK(t, resp)
	})

	// ── TEST: Unknown header value uses default route ─────────
	t.Run("unknown_header_uses_default", func(t *testing.T) {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/api/info",
			Headers: map[string]string{
				"x-version": "stable",
			},
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		assert.NoError(t, err)
		assert.StatusOK(t, resp)
	})
}

// TestWaypointDeployed verifies that a waypoint proxy is running in the
// test namespace when the waypoint Gateway resource is applied.
func TestWaypointDeployed(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := manifest.TemplateValues{
		Namespace:        env.Namespaces.HTTP,
		TestID:           env.RunID,
		GatewayClassName: "istio-waypoint",
	}

	// Apply just the waypoint
	result, err := applier.ApplyFile(ctx, testdataFS,
		"testdata/waypoint.yaml", values)
	if err != nil {
		t.Fatalf("apply waypoint: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	// Wait for waypoint deployment to be created and ready
	time.Sleep(10 * time.Second)
	assert.WaypointRunning(ctx, t, env.Clientset, env.Namespaces.HTTP)
}
