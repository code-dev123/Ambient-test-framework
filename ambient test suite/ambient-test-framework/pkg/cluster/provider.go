// pkg/cluster/provider.go
package cluster

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ConnectOpts holds connection parameters for a cluster provider.
type ConnectOpts struct {
	Kubeconfig  string // path to kubeconfig file; "" means in-cluster or default
	ClusterName string // cloud-provider cluster name (e.g., EKS cluster name)
	Region      string // cloud-provider region
}

// ClusterStatus holds the result of a pre-flight health check.
type ClusterStatus struct {
	AmbientEnabled    bool
	WaypointInstalled bool
	IstioVersion      string
	NodeCount         int
}

// ClusterProvider is the interface every cluster backend must implement.
type ClusterProvider interface {
	// Connect returns Kubernetes clients for the cluster.
	Connect(ctx context.Context, opts ConnectOpts) (
		kubernetes.Interface, dynamic.Interface, error)

	// Healthcheck validates that the cluster is suitable for ambient tests.
	Healthcheck(ctx context.Context) (*ClusterStatus, error)
}
