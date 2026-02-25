// pkg/manifest/wait.go
package manifest

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitForReady polls until all resources in an ApplyResult
// reach their expected ready state.
func (a *Applier) WaitForReady(ctx context.Context,
	result *ApplyResult, timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx,
		2*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			for _, r := range result.Resources {
				ready, err := a.isResourceReady(ctx, r)
				if err != nil || !ready {
					return false, nil
				}
			}
			return true, nil
		})
}

func (a *Applier) isResourceReady(ctx context.Context,
	r AppliedResource) (bool, error) {

	obj, err := a.dynClient.Resource(r.GVR).
		Namespace(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	// Check common readiness patterns
	switch r.GVR.Resource {
	case "gateways":
		return gatewayReady(obj), nil
	case "httproutes", "grpcroutes":
		return routeAccepted(obj), nil
	case "deployments":
		return deploymentReady(obj), nil
	default:
		// For CRDs without status (like AuthzPolicy),
		// existence is sufficient
		return true, nil
	}
}

// gatewayReady checks if a Gateway resource has a programmed condition.
func gatewayReady(obj *unstructured.Unstructured) bool {
	conditions, found, _ := unstructured.NestedSlice(obj.Object,
		"status", "conditions")
	if !found {
		return false
	}
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == "Programmed" && cond["status"] == "True" {
			return true
		}
	}
	return false
}

// routeAccepted checks if an HTTPRoute/GRPCRoute has been accepted.
func routeAccepted(obj *unstructured.Unstructured) bool {
	parents, found, _ := unstructured.NestedSlice(obj.Object,
		"status", "parents")
	if !found {
		return false
	}
	for _, p := range parents {
		parent, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		conditions, _, _ := unstructured.NestedSlice(parent, "conditions")
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if cond["type"] == "Accepted" && cond["status"] == "True" {
				return true
			}
		}
	}
	return false
}

// deploymentReady checks if a Deployment has all replicas available.
func deploymentReady(obj *unstructured.Unstructured) bool {
	desired, found, _ := unstructured.NestedInt64(obj.Object,
		"spec", "replicas")
	if !found {
		return false
	}
	available, _, _ := unstructured.NestedInt64(obj.Object,
		"status", "availableReplicas")

	if desired == 0 {
		return true
	}
	if available < desired {
		return false
	}
	return true
}

// WaitForDeletion polls until the given resource no longer exists.
func (a *Applier) WaitForDeletion(ctx context.Context,
	r AppliedResource, timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx,
		2*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			_, err := a.dynClient.Resource(r.GVR).
				Namespace(r.Namespace).
				Get(ctx, r.Name, metav1.GetOptions{})
			if err != nil {
				// Resource is gone
				return true, nil
			}
			return false, nil
		})
}

// WaitForResultDeletion waits until all resources in an ApplyResult are deleted.
func (a *Applier) WaitForResultDeletion(ctx context.Context,
	result *ApplyResult, timeout time.Duration) error {

	if result == nil {
		return nil
	}
	for _, r := range result.Resources {
		if err := a.WaitForDeletion(ctx, r, timeout); err != nil {
			return fmt.Errorf("waiting for deletion of %s/%s: %w",
				r.Namespace, r.Name, err)
		}
	}
	return nil
}
