// test/baseline/no_mesh/no_mesh_test.go
package no_mesh

import (
	"context"
	"net/http"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yourorg/ambient-test-framework/pkg/assert"
	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	"github.com/yourorg/ambient-test-framework/test"
)

// TestNoMeshBaselineHTTP establishes the baseline: HTTP traffic works
// in the non-mesh namespace without any Istio policy applied.
func TestNoMeshBaselineHTTP(t *testing.T) {
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}

	endpoint := traffic.Endpoint{
		Host: "podinfo." + env.Namespaces.NonMesh + ".svc.cluster.local",
		Port: 9898,
		Path: "/healthz",
	}

	// ── TEST: Plain HTTP GET succeeds in non-mesh namespace ───
	t.Run("plain_http_succeeds", func(t *testing.T) {
		assert.EventuallyHTTPOK(ctx, t, client, endpoint,
			2*time.Second, 30*time.Second)
	})

	// ── TEST: Multiple requests all succeed ───────────────────
	t.Run("multiple_requests_succeed", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			resp, err := traffic.HTTPGet(ctx, client, endpoint)
			assert.NoError(t, err)
			assert.StatusOK(t, resp)
		}
	})
}

// TestNoMeshBaselineTCP establishes the baseline: TCP connectivity works
// in the non-mesh namespace.
func TestNoMeshBaselineTCP(t *testing.T) {
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	endpoint := traffic.TCPEndpoint{
		Host: "podinfo." + env.Namespaces.NonMesh + ".svc.cluster.local",
		Port: 9898,
	}

	// ── TEST: TCP connection succeeds ────────────────────────
	t.Run("tcp_connect_succeeds", func(t *testing.T) {
		err := traffic.TCPConnectWithRetry(ctx, endpoint,
			2*time.Second, 30*time.Second)
		assert.NoError(t, err)
	})
}

// TestNoMeshIsolation verifies that the non-mesh namespace does NOT
// have the ambient mode label — this is the control group.
func TestNoMeshIsolation(t *testing.T) {
	t.Parallel()

	env := test.GetEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Run("non_mesh_namespace_not_ambient", func(t *testing.T) {
		ns, err := env.Clientset.CoreV1().Namespaces().Get(ctx,
			env.Namespaces.NonMesh, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get namespace %s: %v", env.Namespaces.NonMesh, err)
		}
		if v, ok := ns.Labels["istio.io/dataplane-mode"]; ok && v == "ambient" {
			t.Errorf("non-mesh namespace %s should NOT have ambient label, but does",
				env.Namespaces.NonMesh)
		}
	})

	t.Run("ztunnel_running", func(t *testing.T) {
		assert.ZTunnelRunning(ctx, t, env.Clientset)
	})
}
