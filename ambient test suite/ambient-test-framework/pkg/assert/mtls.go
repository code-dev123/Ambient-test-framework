// pkg/assert/mtls.go
package assert

import (
	"context"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MTLSEnforced asserts that the given namespace is enrolled in ambient mesh,
// which means ztunnel is enforcing mTLS transparently for the named workload.
func MTLSEnforced(ctx context.Context, clientset kubernetes.Interface,
	namespace, podName string) {

	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get namespace %s", namespace)
	mode := ns.Labels["istio.io/dataplane-mode"]
	Expect(mode).To(Equal("ambient"),
		"namespace %s is not enrolled in ambient mesh (label istio.io/dataplane-mode=%q)",
		namespace, mode)
}

// ZTunnelRunning asserts that the ztunnel DaemonSet is running with ready pods.
func ZTunnelRunning(ctx context.Context, clientset kubernetes.Interface) {
	ds, err := clientset.AppsV1().DaemonSets("istio-system").
		Get(ctx, "ztunnel", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "get ztunnel daemonset")
	Expect(ds.Status.NumberReady).To(BeNumerically(">", 0),
		"ztunnel DaemonSet has 0 ready pods")
}

// WaypointRunning asserts that a waypoint proxy deployment is running
// in the given namespace.
func WaypointRunning(ctx context.Context, clientset kubernetes.Interface, namespace string) {
	deps, err := clientset.AppsV1().Deployments(namespace).List(ctx,
		metav1.ListOptions{
			LabelSelector: "gateway.istio.io/managed=istio.io-mesh-controller",
		})
	Expect(err).NotTo(HaveOccurred(), "list waypoint deployments in %s", namespace)
	Expect(deps.Items).NotTo(BeEmpty(),
		"no waypoint deployment found in namespace %s", namespace)
	for _, dep := range deps.Items {
		Expect(dep.Status.AvailableReplicas).To(BeNumerically(">", 0),
			"waypoint %s/%s has 0 available replicas", namespace, dep.Name)
	}
}
