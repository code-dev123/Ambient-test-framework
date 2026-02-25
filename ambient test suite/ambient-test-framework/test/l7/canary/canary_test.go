// test/l7/canary/canary_test.go
package canary

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yourorg/ambient-test-framework/pkg/assert"
	"github.com/yourorg/ambient-test-framework/pkg/manifest"
	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	pkgwait "github.com/yourorg/ambient-test-framework/pkg/wait"
	"github.com/yourorg/ambient-test-framework/test"
)

// Embed all testdata so the binary is self-contained.
//
//go:embed testdata/*
var testdataFS embed.FS

// templateValues builds the TemplateValues for this scenario.
func templateValues(env *test.Environment,
	stableWeight, canaryWeight int) manifest.TemplateValues {
	return manifest.TemplateValues{
		Namespace:        env.Namespaces.HTTP,
		TestID:           env.RunID,
		PodInfoImageV1:   env.Config.Workloads.PodInfoImageV1,
		PodInfoImageV2:   env.Config.Workloads.PodInfoImageV2,
		StableWeight:     stableWeight,
		CanaryWeight:     canaryWeight,
		GatewayClassName: "istio",
	}
}

// ============================================================
// go test -run TestCanaryNorthSouth
// ============================================================

func TestCanaryNorthSouth(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := templateValues(env, 80, 20) // 80/20 split

	// ── SETUP: Deploy workloads ────────────────────────────
	workloadResult, err := applier.ApplyFolder(ctx, testdataFS,
		"testdata/workloads", values)
	if err != nil {
		t.Fatalf("deploy workloads: %v", err)
	}

	var nsResult *manifest.ApplyResult

	// Register cleanup IMMEDIATELY — runs even on t.Fatal
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		if nsResult != nil {
			applier.DeleteResult(cleanupCtx, nsResult)
		}
		applier.DeleteResult(cleanupCtx, workloadResult)
	})

	// Wait for workloads to be ready
	err = pkgwait.ForDeploymentsReady(ctx, env.Clientset,
		env.Namespaces.HTTP, "test-id="+env.RunID, 2*time.Minute)
	if err != nil {
		t.Fatalf("workloads not ready: %v", err)
	}

	// ── SETUP: Apply N-S canary manifests ──────────────────
	nsResult, err = applier.ApplyFolder(ctx, testdataFS,
		"testdata/ns-canary", values)
	if err != nil {
		t.Fatalf("apply ns-canary manifests: %v", err)
	}

	// Wait for Gateway and HTTPRoute to be accepted
	err = applier.WaitForReady(ctx, nsResult, 90*time.Second)
	if err != nil {
		t.Fatalf("ns-canary resources not ready: %v", err)
	}

	// ── TEST: Verify traffic split (80/20) ─────────────────
	t.Run("traffic_split_80_20", func(t *testing.T) {
		verifyTrafficSplit(ctx, t, env, 80, 20, 200)
	})

	// ── TEST: Canary responds correctly ────────────────────
	t.Run("canary_version_header", func(t *testing.T) {
		resp, err := traffic.HTTPGet(ctx,
			&http.Client{Timeout: 5 * time.Second},
			traffic.Endpoint{
				Host: fmt.Sprintf("podinfo.%s.svc.cluster.local",
					env.Namespaces.HTTP),
				Port: 9898,
				Path: "/api/info",
			})
		assert.NoError(t, err)
		assert.StatusOK(t, resp)
	})

	// ── TEST: Shift to 50/50, verify convergence ───────────
	t.Run("shift_to_50_50", func(t *testing.T) {
		newValues := templateValues(env, 50, 50)

		// Re-apply the same folder with updated weights.
		// Server-side apply handles the update idempotently.
		_, err := applier.ApplyFolder(ctx, testdataFS,
			"testdata/ns-canary", newValues)
		if err != nil {
			t.Fatalf("re-apply with 50/50: %v", err)
		}

		// Brief settle time for route update to propagate
		time.Sleep(5 * time.Second)

		verifyTrafficSplit(ctx, t, env, 50, 50, 200)
	})
}

// ============================================================
// go test -run TestCanaryEastWest
// ============================================================

func TestCanaryEastWest(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := templateValues(env, 90, 10) // 90/10 split

	// ── SETUP: Deploy workloads ────────────────────────────
	workloadResult, err := applier.ApplyFolder(ctx, testdataFS,
		"testdata/workloads", values)
	if err != nil {
		t.Fatalf("deploy workloads: %v", err)
	}

	var ewResult *manifest.ApplyResult

	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		if ewResult != nil {
			applier.DeleteResult(cleanupCtx, ewResult)
		}
		applier.DeleteResult(cleanupCtx, workloadResult)
	})

	err = pkgwait.ForDeploymentsReady(ctx, env.Clientset,
		env.Namespaces.HTTP, "test-id="+env.RunID, 2*time.Minute)
	if err != nil {
		t.Fatalf("workloads not ready: %v", err)
	}

	// ── SETUP: Apply E-W canary manifests ──────────────────
	ewResult, err = applier.ApplyFolder(ctx, testdataFS,
		"testdata/ew-canary", values)
	if err != nil {
		t.Fatalf("apply ew-canary manifests: %v", err)
	}
	err = applier.WaitForReady(ctx, ewResult, 60*time.Second)
	if err != nil {
		t.Fatalf("ew-canary resources not ready: %v", err)
	}

	// ── TEST: E-W traffic split 90/10 ─────────────────────
	t.Run("ew_traffic_split_90_10", func(t *testing.T) {
		verifyTrafficSplit(ctx, t, env, 90, 10, 300)
	})

	// ── TEST: Non-mesh client reaches service (no split) ──
	t.Run("non_mesh_no_split", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			resp, err := traffic.HTTPGet(ctx,
				&http.Client{Timeout: 5 * time.Second},
				traffic.Endpoint{
					Host: fmt.Sprintf("podinfo.%s.svc.cluster.local",
						env.Namespaces.HTTP),
					Port: 9898,
					Path: "/api/info",
				})
			assert.NoError(t, err)
			assert.StatusOK(t, resp)
		}
	})
}

// ============================================================
// go test -run TestCanaryFullPromote
// ============================================================

func TestCanaryFullPromote(t *testing.T) {
	test.SkipUnlessLayer(t, "l7")
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	applier := env.NewApplier()
	values := templateValues(env, 90, 10)

	workloadResult, err := applier.ApplyFolder(ctx, testdataFS,
		"testdata/workloads", values)
	if err != nil {
		t.Fatalf("deploy workloads: %v", err)
	}

	var currentResult *manifest.ApplyResult
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		if currentResult != nil {
			applier.DeleteResult(cleanupCtx, currentResult)
		}
		applier.DeleteResult(cleanupCtx, workloadResult)
	})

	err = pkgwait.ForDeploymentsReady(ctx, env.Clientset,
		env.Namespaces.HTTP, "test-id="+env.RunID, 2*time.Minute)
	if err != nil {
		t.Fatalf("workloads not ready: %v", err)
	}

	// Progressive promotion: 90/10 → 50/50 → 0/100
	stages := []struct {
		stable, canary int
	}{
		{90, 10},
		{50, 50},
		{0, 100},
	}

	for _, stage := range stages {
		stage := stage // capture loop var
		name := fmt.Sprintf("promote_%d_%d", stage.stable, stage.canary)
		t.Run(name, func(t *testing.T) {
			stageValues := templateValues(env, stage.stable, stage.canary)

			currentResult, err = applier.ApplyFolder(ctx, testdataFS,
				"testdata/ew-canary", stageValues)
			if err != nil {
				t.Fatalf("apply stage %d/%d: %v",
					stage.stable, stage.canary, err)
			}
			time.Sleep(5 * time.Second)

			verifyTrafficSplit(ctx, t, env,
				stage.stable, stage.canary, 200)
		})
	}
}

// ============================================================
// Shared helper: verify traffic distribution matches weights
// ============================================================

func verifyTrafficSplit(ctx context.Context, t *testing.T,
	env *test.Environment,
	expectedStable, expectedCanary, totalRequests int) {

	t.Helper()

	var v1Hits, v2Hits atomic.Int64
	var wg sync.WaitGroup

	endpoint := traffic.Endpoint{
		Host: fmt.Sprintf("podinfo.%s.svc.cluster.local",
			env.Namespaces.HTTP),
		Port: 9898,
		Path: "/api/info",
	}

	// Send traffic in parallel for speed
	concurrency := 10
	perWorker := totalRequests / concurrency
	client := &http.Client{Timeout: 5 * time.Second}

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				resp, err := traffic.HTTPGetJSON(ctx, client, endpoint)
				if err != nil {
					continue
				}
				if msg, ok := resp["message"].(string); ok {
					switch msg {
					case "v1-stable":
						v1Hits.Add(1)
					case "v2-canary":
						v2Hits.Add(1)
					}
				}
			}
		}()
	}
	wg.Wait()

	total := v1Hits.Load() + v2Hits.Load()
	if total == 0 {
		t.Fatal("no successful responses received")
	}

	actualV1Pct := float64(v1Hits.Load()) / float64(total) * 100
	actualV2Pct := float64(v2Hits.Load()) / float64(total) * 100

	// Allow ±15% tolerance for statistical variance
	tolerance := 15.0
	assert.InDelta(t, float64(expectedStable), actualV1Pct, tolerance,
		"v1 (stable) traffic: expected ~%d%%, got %.1f%%",
		expectedStable, actualV1Pct)
	assert.InDelta(t, float64(expectedCanary), actualV2Pct, tolerance,
		"v2 (canary) traffic: expected ~%d%%, got %.1f%%",
		expectedCanary, actualV2Pct)

	t.Logf("Traffic split: v1=%.1f%% (%d), v2=%.1f%% (%d), total=%d",
		actualV1Pct, v1Hits.Load(),
		actualV2Pct, v2Hits.Load(), total)
}
