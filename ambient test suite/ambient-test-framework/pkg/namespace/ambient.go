// pkg/namespace/ambient.go
package namespace

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	// AmbientModeLabel is the label that enrolls a namespace in ambient mesh.
	AmbientModeLabel = "istio.io/dataplane-mode"
	// AmbientModeValue is the value for the ambient mode label.
	AmbientModeValue = "ambient"
	// WaypointLabel marks a namespace to use a waypoint proxy for L7.
	WaypointLabel = "istio.io/use-waypoint"
)

// LabelForAmbient adds the ambient mesh label to a namespace.
func LabelForAmbient(ctx context.Context,
	clientset kubernetes.Interface, namespaceName string) error {

	return patchNamespaceLabels(ctx, clientset, namespaceName,
		map[string]string{AmbientModeLabel: AmbientModeValue})
}

// UnlabelAmbient removes the ambient mesh label from a namespace.
func UnlabelAmbient(ctx context.Context,
	clientset kubernetes.Interface, namespaceName string) error {

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				AmbientModeLabel: nil,
			},
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Namespaces().Patch(ctx, namespaceName,
		types.MergePatchType, data, metav1.PatchOptions{})
	return err
}

// SetWaypoint sets the waypoint name for a namespace.
func SetWaypoint(ctx context.Context,
	clientset kubernetes.Interface, namespaceName, waypointName string) error {

	return patchNamespaceLabels(ctx, clientset, namespaceName,
		map[string]string{WaypointLabel: waypointName})
}

func patchNamespaceLabels(ctx context.Context,
	clientset kubernetes.Interface, namespaceName string,
	labels map[string]string) error {

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": labels,
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal patch: %w", err)
	}
	_, err = clientset.CoreV1().Namespaces().Patch(ctx, namespaceName,
		types.MergePatchType, data, metav1.PatchOptions{})
	return err
}
