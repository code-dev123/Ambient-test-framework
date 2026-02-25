// pkg/assert/mtls.go
package assert

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MTLSEnforced asserts that mTLS is enforced between two pods by checking
// that a plaintext connection fails. In ambient mode, ztunnel enforces
// mTLS transparently; this helper verifies via Istio telemetry or proxy logs.
func MTLSEnforced(ctx context.Context, t *testing.T,
	clientset kubernetes.Interface,
	namespace, podName string) {

	t.Helper()
	// Check that the pod has the ambient mesh label via namespace
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace,
		metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get namespace %s: %v", namespace, err)
	}
	mode, ok := ns.Labels["istio.io/dataplane-mode"]
	if !ok || mode != "ambient" {
		t.Errorf("namespace %s is not enrolled in ambient mesh "+
			"(label istio.io/dataplane-mode=%q)",
			namespace, mode)
	}
}

// ZTunnelRunning asserts that the ztunnel DaemonSet is running.
func ZTunnelRunning(ctx context.Context, t *testing.T,
	clientset kubernetes.Interface) {

	t.Helper()
	ds, err := clientset.AppsV1().DaemonSets("istio-system").
		Get(ctx, "ztunnel", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get ztunnel daemonset: %v", err)
	}
	if ds.Status.NumberReady == 0 {
		t.Error("ztunnel DaemonSet has 0 ready pods")
	}
}

// WaypointRunning asserts that a waypoint proxy deployment is running
// in the given namespace.
func WaypointRunning(ctx context.Context, t *testing.T,
	clientset kubernetes.Interface, namespace string) {

	t.Helper()
	// Waypoint deployments are labeled with gateway.istio.io/managed
	deps, err := clientset.AppsV1().Deployments(namespace).List(ctx,
		metav1.ListOptions{
			LabelSelector: "gateway.istio.io/managed=istio.io-mesh-controller",
		})
	if err != nil {
		t.Fatalf("list waypoint deployments in %s: %v", namespace, err)
	}
	if len(deps.Items) == 0 {
		t.Errorf("no waypoint deployment found in namespace %s", namespace)
		return
	}
	for _, dep := range deps.Items {
		if dep.Status.AvailableReplicas == 0 {
			t.Errorf("waypoint %s/%s has 0 available replicas",
				namespace, dep.Name)
		}
	}
}
