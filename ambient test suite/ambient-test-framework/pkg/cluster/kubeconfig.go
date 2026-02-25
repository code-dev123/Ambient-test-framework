// pkg/cluster/kubeconfig.go
package cluster

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// BuildClientsFromKubeconfig builds Kubernetes clients from a kubeconfig path.
// If kubeconfigPath is empty it falls back to KUBECONFIG env var, then
// ~/.kube/config, then in-cluster config.
func BuildClientsFromKubeconfig(kubeconfigPath string) (
	kubernetes.Interface, dynamic.Interface, *rest.Config, error) {

	cfg, err := loadRestConfig(kubeconfigPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("build rest config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("build clientset: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("build dynamic client: %w", err)
	}

	return clientset, dynClient, cfg, nil
}

func loadRestConfig(kubeconfigPath string) (*rest.Config, error) {
	// 1. Explicit path
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	// 2. KUBECONFIG env var
	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		return clientcmd.BuildConfigFromFlags("", kc)
	}

	// 3. Default ~/.kube/config
	home, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, statErr := os.Stat(defaultPath); statErr == nil {
			return clientcmd.BuildConfigFromFlags("", defaultPath)
		}
	}

	// 4. In-cluster config
	return rest.InClusterConfig()
}
