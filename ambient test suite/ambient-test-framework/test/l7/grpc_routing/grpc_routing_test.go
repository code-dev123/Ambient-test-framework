// test/l7/grpc_routing/grpc_routing_test.go
package grpc_routing

import (
	"context"
	"embed"
	"testing"
	"time"

	"github.com/yourorg/ambient-test-framework/pkg/manifest"
	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	"github.com/yourorg/ambient-test-framework/test"
)

//go:embed testdata/*
var testdataFS embed.FS

// TestGRPCRouteBasic verifies that a GRPCRoute is accepted by the waypoint
// and that gRPC traffic reaches the backend.
func TestGRPCRouteBasic(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := manifest.TemplateValues{
		Namespace:        env.Namespaces.GRPC,
		TestID:           env.RunID,
		GatewayClassName: "istio-waypoint",
	}

	// ── SETUP ────────────────────────────────────────────────
	result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
	if err != nil {
		t.Fatalf("apply grpc-routing manifests: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	err = applier.WaitForReady(ctx, result, 90*time.Second)
	if err != nil {
		t.Fatalf("grpc-routing resources not ready: %v", err)
	}

	endpoint := traffic.GRPCEndpoint{
		Host:    "podinfo-grpc." + env.Namespaces.GRPC + ".svc.cluster.local",
		Port:    9999,
		Service: "",
	}

	// ── TEST: gRPC health check succeeds ─────────────────────
	t.Run("grpc_health_check", func(t *testing.T) {
		if err := traffic.GRPCHealthCheck(ctx, endpoint); err != nil {
			t.Errorf("gRPC health check failed: %v", err)
		}
	})
}

// TestGRPCRouteTLS verifies that gRPC over mTLS (via ztunnel) works
// when the namespace is enrolled in ambient mesh.
func TestGRPCRouteTLS(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	endpoint := traffic.GRPCEndpoint{
		Host:    "podinfo-grpc." + env.Namespaces.GRPC + ".svc.cluster.local",
		Port:    9999,
		Service: "",
	}

	// ── TEST: gRPC works through ambient mTLS ─────────────────
	t.Run("grpc_through_ztunnel", func(t *testing.T) {
		if err := traffic.GRPCHealthCheck(ctx, endpoint); err != nil {
			t.Errorf("gRPC health check through ztunnel failed: %v", err)
		}
	})

	// ── TEST: gRPC connection can be established ──────────────
	t.Run("grpc_dial_succeeds", func(t *testing.T) {
		conn, err := traffic.GRPCDial(ctx, endpoint)
		if err != nil {
			t.Fatalf("gRPC dial failed: %v", err)
		}
		conn.Close()
	})
}
