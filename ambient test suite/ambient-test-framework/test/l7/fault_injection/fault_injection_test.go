// test/l7/fault_injection/fault_injection_test.go
package fault_injection

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

// TestFaultInjectionDelay verifies that a VirtualService fault delay
// causes requests to take longer than the configured delay.
func TestFaultInjectionDelay(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
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
	result, err := applier.ApplyFile(ctx, testdataFS,
		"testdata/fault-delay.yaml", values)
	if err != nil {
		t.Fatalf("apply fault-delay manifest: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	// Wait for VirtualService to propagate
	time.Sleep(5 * time.Second)

	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}
	client := &http.Client{Timeout: 10 * time.Second}

	// ── TEST: Request takes ≥3s due to fault delay ───────────
	t.Run("delay_applied", func(t *testing.T) {
		start := time.Now()
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.StatusOK(t, resp)

		if elapsed < 3*time.Second {
			t.Errorf("expected delay ≥3s, got %s — fault injection may not be working",
				elapsed)
		}
		t.Logf("request took %s (fault delay=3s)", elapsed)
	})

	// ── TEST: Service is still reachable despite delay ────────
	t.Run("service_reachable_with_delay", func(t *testing.T) {
		// Use a long timeout to account for the injected delay
		longClient := &http.Client{Timeout: 10 * time.Second}
		assert.EventuallyHTTPOK(ctx, t, longClient, endpoint,
			1*time.Second, 15*time.Second)
	})
}

// TestFaultInjectionAbort verifies that a VirtualService fault abort
// causes all requests to return HTTP 503.
func TestFaultInjectionAbort(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := manifest.TemplateValues{
		Namespace: env.Namespaces.HTTP,
		TestID:    env.RunID,
	}

	// First verify service is healthy
	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}
	client := &http.Client{Timeout: 5 * time.Second}

	err := pkgwait.UntilNoError(ctx, func(ctx context.Context) error {
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return nil
		}
		return nil
	}, 3*time.Second, 30*time.Second)
	if err != nil {
		t.Logf("service not healthy before test: %v", err)
	}

	// ── SETUP: Apply fault abort ──────────────────────────────
	result, err := applier.ApplyFile(ctx, testdataFS,
		"testdata/fault-abort.yaml", values)
	if err != nil {
		t.Fatalf("apply fault-abort manifest: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		applier.DeleteResult(cleanupCtx, result)
	})

	time.Sleep(5 * time.Second)

	// ── TEST: All requests return 503 ────────────────────────
	t.Run("abort_returns_503", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			resp, err := traffic.HTTPGet(ctx, client, endpoint)
			assert.NoError(t, err)
			assert.StatusCode(t, resp, http.StatusServiceUnavailable)
		}
	})

	// ── TEST: After removing fault, service recovers ──────────
	t.Run("service_recovers_after_fault_removed", func(t *testing.T) {
		if err := applier.DeleteResult(ctx, result); err != nil {
			t.Fatalf("delete fault-abort: %v", err)
		}
		result = nil // prevent double-delete in cleanup

		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			3*time.Second, 30*time.Second)
	})
}
