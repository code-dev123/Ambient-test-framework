// pkg/workload/deployer.go
package workload

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/yourorg/ambient-test-framework/pkg/namespace"
)

// WorkloadSet holds the client pods deployed by the test suite.
type WorkloadSet struct {
	HTTPClient    *PodRef // mesh-enrolled HTTP client
	GRPCClient    *PodRef // mesh-enrolled gRPC client
	NonMeshClient *PodRef // non-mesh HTTP client
}

// PodRef identifies a pod that can be used to generate traffic.
type PodRef struct {
	Namespace string
	Name      string
	// ServiceIP is the cluster IP of the backing service (for direct HTTP)
	ServiceIP string
}

// PodInfoDeployer deploys podinfo workloads as traffic clients.
type PodInfoDeployer struct {
	clientset kubernetes.Interface
}

// NewPodInfoDeployer creates a new PodInfoDeployer.
func NewPodInfoDeployer(clientset kubernetes.Interface) *PodInfoDeployer {
	return &PodInfoDeployer{clientset: clientset}
}

// DeployClients deploys a client pod in each test namespace and waits
// until they are ready.
func (d *PodInfoDeployer) DeployClients(ctx context.Context,
	ns *namespace.NamespaceSet) (*WorkloadSet, error) {

	wl := &WorkloadSet{}

	// HTTP client in the ambient-enrolled namespace
	httpRef, err := d.deployClient(ctx, ns.HTTP, "http-client",
		"ghcr.io/stefanprodan/podinfo:6.5.0")
	if err != nil {
		return nil, fmt.Errorf("deploy http client: %w", err)
	}
	wl.HTTPClient = httpRef

	// gRPC client in the ambient-enrolled namespace
	grpcRef, err := d.deployClient(ctx, ns.GRPC, "grpc-client",
		"ghcr.io/stefanprodan/podinfo:6.5.0")
	if err != nil {
		return nil, fmt.Errorf("deploy grpc client: %w", err)
	}
	wl.GRPCClient = grpcRef

	// Non-mesh client
	nonMeshRef, err := d.deployClient(ctx, ns.NonMesh, "non-mesh-client",
		"ghcr.io/stefanprodan/podinfo:6.5.0")
	if err != nil {
		return nil, fmt.Errorf("deploy non-mesh client: %w", err)
	}
	wl.NonMeshClient = nonMeshRef

	return wl, nil
}

func (d *PodInfoDeployer) deployClient(ctx context.Context,
	namespaceName, name, image string) (*PodRef, error) {

	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "client",
							Image: image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9898, Name: "http"},
							},
						},
					},
				},
			},
		},
	}

	_, err := d.clientset.AppsV1().Deployments(namespaceName).
		Create(ctx, dep, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("create deployment %s/%s: %w",
			namespaceName, name, err)
	}

	// Wait for the deployment to be ready
	if err := d.waitForDeployment(ctx, namespaceName, name,
		2*time.Minute); err != nil {
		return nil, fmt.Errorf("deployment %s not ready: %w", name, err)
	}

	return &PodRef{
		Namespace: namespaceName,
		Name:      name,
	}, nil
}

func (d *PodInfoDeployer) waitForDeployment(ctx context.Context,
	namespaceName, name string, timeout time.Duration) error {

	return wait.PollUntilContextTimeout(ctx, 3*time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			dep, err := d.clientset.AppsV1().Deployments(namespaceName).
				Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if dep.Status.AvailableReplicas >= *dep.Spec.Replicas {
				return true, nil
			}
			return false, nil
		})
}
