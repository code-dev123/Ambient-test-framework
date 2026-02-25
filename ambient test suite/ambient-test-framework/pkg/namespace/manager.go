// pkg/namespace/manager.go
package namespace

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Config holds namespace creation settings.
type Config struct {
	HTTPPrefix    string // prefix for the HTTP test namespace
	GRPCPrefix    string // prefix for the gRPC test namespace
	NonMeshPrefix string // prefix for the non-mesh test namespace
	Suffix        string // unique suffix (usually the run ID)
	AmbientLabel  bool   // whether to label namespaces for ambient mode
}

// NamespaceSet holds the names of all test namespaces.
type NamespaceSet struct {
	HTTP    string
	GRPC    string
	NonMesh string
}

// Manager creates and deletes test namespaces.
type Manager struct {
	clientset kubernetes.Interface
}

// NewManager creates a new namespace Manager.
func NewManager(clientset kubernetes.Interface) *Manager {
	return &Manager{clientset: clientset}
}

// CreateTestNamespaces creates the namespaces defined by cfg and returns
// the resulting NamespaceSet.
func (m *Manager) CreateTestNamespaces(ctx context.Context,
	cfg Config) (*NamespaceSet, error) {

	ns := &NamespaceSet{
		HTTP:    fmt.Sprintf("%s-%s", cfg.HTTPPrefix, cfg.Suffix),
		GRPC:    fmt.Sprintf("%s-%s", cfg.GRPCPrefix, cfg.Suffix),
		NonMesh: fmt.Sprintf("%s-%s", cfg.NonMeshPrefix, cfg.Suffix),
	}

	for _, name := range []string{ns.HTTP, ns.GRPC} {
		labels := map[string]string{
			"test-suite": "ambient",
			"test-id":    cfg.Suffix,
		}
		if cfg.AmbientLabel {
			// Label to enroll in ambient mesh
			labels["istio.io/dataplane-mode"] = "ambient"
		}
		if err := m.createNamespace(ctx, name, labels); err != nil {
			return nil, err
		}
	}

	// Non-mesh namespace — no ambient label
	nonMeshLabels := map[string]string{
		"test-suite": "ambient",
		"test-id":    cfg.Suffix,
	}
	if err := m.createNamespace(ctx, ns.NonMesh, nonMeshLabels); err != nil {
		return nil, err
	}

	return ns, nil
}

// DeleteAll removes all namespaces in the set.
func (m *Manager) DeleteAll(ctx context.Context, ns *NamespaceSet) {
	for _, name := range []string{ns.HTTP, ns.GRPC, ns.NonMesh} {
		_ = m.clientset.CoreV1().Namespaces().Delete(ctx, name,
			metav1.DeleteOptions{})
	}
}

func (m *Manager) createNamespace(ctx context.Context,
	name string, labels map[string]string) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	_, err := m.clientset.CoreV1().Namespaces().Create(ctx, ns,
		metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %s: %w", name, err)
	}
	return nil
}
