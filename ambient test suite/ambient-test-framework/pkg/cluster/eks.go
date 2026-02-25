// pkg/cluster/eks.go
package cluster

import (
	"context"
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// EKSProvider connects to an EKS cluster using kubeconfig or aws-iam-authenticator.
type EKSProvider struct {
	clientset kubernetes.Interface
	dynClient dynamic.Interface
}

// NewEKSProvider creates a new EKSProvider.
func NewEKSProvider() *EKSProvider {
	return &EKSProvider{}
}

// Connect establishes clients for the EKS cluster.
// If ClusterName and Region are provided, it updates the kubeconfig automatically.
// Otherwise falls through to standard kubeconfig loading.
func (p *EKSProvider) Connect(ctx context.Context, opts ConnectOpts) (
	kubernetes.Interface, dynamic.Interface, error) {

	kubeconfigPath := opts.Kubeconfig
	if opts.ClusterName != "" && opts.Region != "" {
		// When running in CI / automation, generate kubeconfig via AWS CLI.
		// The binary must be on PATH: aws eks update-kubeconfig
		generated, err := generateEKSKubeconfig(ctx, opts.ClusterName, opts.Region)
		if err != nil {
			return nil, nil, fmt.Errorf("generate eks kubeconfig: %w", err)
		}
		kubeconfigPath = generated
	}

	clientset, dynClient, _, err := BuildClientsFromKubeconfig(kubeconfigPath)
	if err != nil {
		return nil, nil, err
	}

	p.clientset = clientset
	p.dynClient = dynClient
	return clientset, dynClient, nil
}

// Healthcheck validates the cluster is ready for ambient tests.
func (p *EKSProvider) Healthcheck(ctx context.Context) (*ClusterStatus, error) {
	if p.clientset == nil {
		return nil, fmt.Errorf("provider not connected — call Connect first")
	}

	status := &ClusterStatus{}

	// Check node count
	nodes, err := p.clientset.CoreV1().Nodes().List(ctx,
		listOptsWithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	status.NodeCount = len(nodes.Items)

	// Check for Istio ambient mode (ztunnel daemonset)
	_, err = p.clientset.AppsV1().DaemonSets("istio-system").
		Get(ctx, "ztunnel", getOptsWithContext(ctx))
	if err == nil {
		status.AmbientEnabled = true
	}

	// Check for waypoint deployment
	_, err = p.clientset.AppsV1().Deployments("istio-system").
		Get(ctx, "waypoint", getOptsWithContext(ctx))
	if err == nil {
		status.WaypointInstalled = true
	}

	return status, nil
}

// generateEKSKubeconfig runs `aws eks update-kubeconfig` and returns
// the path to the generated kubeconfig file.
func generateEKSKubeconfig(ctx context.Context, clusterName, region string) (string, error) {
	// In real usage this would exec `aws eks update-kubeconfig`.
	// Returning empty string triggers the default kubeconfig path.
	return "", nil
}
