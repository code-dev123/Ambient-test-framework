// pkg/wait/conditions.go
package wait

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// ForDeploymentsReady waits until all deployments matching the label selector
// in the given namespace have all replicas available.
func ForDeploymentsReady(ctx context.Context,
	clientset kubernetes.Interface,
	namespace, labelSelector string,
	timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx, 3*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			deps, err := clientset.AppsV1().Deployments(namespace).List(ctx,
				metav1.ListOptions{LabelSelector: labelSelector})
			if err != nil {
				return false, nil
			}
			if len(deps.Items) == 0 {
				return false, nil
			}
			for _, dep := range deps.Items {
				desired := int32(1)
				if dep.Spec.Replicas != nil {
					desired = *dep.Spec.Replicas
				}
				if dep.Status.AvailableReplicas < desired {
					return false, nil
				}
			}
			return true, nil
		})
}

// ForPodsReady waits until all pods matching the label selector are Running.
func ForPodsReady(ctx context.Context,
	clientset kubernetes.Interface,
	namespace, labelSelector string,
	count int,
	timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx, 3*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			pods, err := clientset.CoreV1().Pods(namespace).List(ctx,
				metav1.ListOptions{LabelSelector: labelSelector})
			if err != nil {
				return false, nil
			}
			ready := 0
			for _, pod := range pods.Items {
				for _, cond := range pod.Status.Conditions {
					if cond.Type == "Ready" && cond.Status == "True" {
						ready++
					}
				}
			}
			return ready >= count, nil
		})
}

// ForNamespaceReady waits until a namespace exists and is Active.
func ForNamespaceReady(ctx context.Context,
	clientset kubernetes.Interface,
	name string,
	timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			ns, err := clientset.CoreV1().Namespaces().Get(ctx, name,
				metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			return ns.Status.Phase == "Active", nil
		})
}

// ForServiceEndpoints waits until a service has at least one endpoint.
func ForServiceEndpoints(ctx context.Context,
	clientset kubernetes.Interface,
	namespace, serviceName string,
	timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx, 3*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			ep, err := clientset.CoreV1().Endpoints(namespace).Get(ctx,
				serviceName, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			for _, subset := range ep.Subsets {
				if len(subset.Addresses) > 0 {
					return true, nil
				}
			}
			return false, nil
		})
}

// ForDaemonSetReady waits until a DaemonSet has all desired pods ready.
func ForDaemonSetReady(ctx context.Context,
	clientset kubernetes.Interface,
	namespace, name string,
	timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx, 3*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx,
				name, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			desired := ds.Status.DesiredNumberScheduled
			if desired == 0 {
				return false, fmt.Errorf("daemonset %s/%s has 0 desired pods",
					namespace, name)
			}
			return ds.Status.NumberReady >= desired, nil
		})
}
