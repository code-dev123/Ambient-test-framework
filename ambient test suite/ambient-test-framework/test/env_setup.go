// test/env_setup.go
package test

import (
	"context"
	"flag"
	"log/slog"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/pkg/cluster"
	"github.com/yourorg/ambient-test-framework/pkg/id"
	"github.com/yourorg/ambient-test-framework/pkg/namespace"
	"github.com/yourorg/ambient-test-framework/pkg/workload"
)

// Test-level CLI flags shared across all test packages.
var (
	FlagKeepNS     = flag.Bool("keep-ns", false, "preserve namespaces after test run")
	FlagKubeconfig = flag.String("kubeconfig", "", "kubeconfig path")
	FlagEKSCluster = flag.String("eks-cluster", "", "EKS cluster name")
	FlagRegion     = flag.String("region", "us-west-2", "AWS region")
)

var (
	sharedNsMgr *namespace.Manager
	sharedNS    *namespace.NamespaceSet
)

// SetupEnvironment initializes the full test environment: cluster connection,
// healthcheck, namespace creation, and workload deployment.
// Call this from BeforeSuite in each test package.
func SetupEnvironment() {
	ctx := context.Background()
	runID := id.Short()
	logger := slog.Default()

	// 1. Connect to cluster
	provider := cluster.NewEKSProvider()
	clientset, dynClient, err := provider.Connect(ctx, cluster.ConnectOpts{
		Kubeconfig:  *FlagKubeconfig,
		ClusterName: *FlagEKSCluster,
		Region:      *FlagRegion,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "cluster connect failed")

	// 2. Pre-flight healthcheck
	status, err := provider.Healthcheck(ctx)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "pre-flight healthcheck failed")
	gomega.Expect(status.AmbientEnabled).To(gomega.BeTrue(),
		"ambient mesh not enabled on cluster — ztunnel DaemonSet not found")

	ginkgo.GinkgoWriter.Printf("cluster connected: nodes=%d ambient=%v waypoint=%v run_id=%s\n",
		status.NodeCount, status.AmbientEnabled, status.WaypointInstalled, runID)

	// 3. Create test namespaces
	sharedNsMgr = namespace.NewManager(clientset)
	ns, err := sharedNsMgr.CreateTestNamespaces(ctx, namespace.Config{
		HTTPPrefix:    "http",
		GRPCPrefix:    "grpc",
		NonMeshPrefix: "non-mesh",
		Suffix:        runID,
		AmbientLabel:  true,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "namespace creation failed")
	sharedNS = ns

	ginkgo.GinkgoWriter.Printf("test namespaces created: http=%s grpc=%s non-mesh=%s\n",
		ns.HTTP, ns.GRPC, ns.NonMesh)

	// 4. Deploy baseline client workloads (traffic sources)
	deployer := workload.NewPodInfoDeployer(clientset)
	wl, err := deployer.DeployClients(ctx, ns)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "workload deploy failed")

	// 5. Store the shared environment for all specs
	SetEnvironment(&Environment{
		Clientset:       clientset,
		DynClient:       dynClient,
		DiscoveryClient: clientset,
		Namespaces:      *ns,
		Workloads:       *wl,
		RunID:           runID,
		Config:          LoadConfig(),
		Logger:          logger,
	})
}

// TeardownEnvironment deletes test namespaces unless --keep-ns is set.
// Call this from AfterSuite in each test package.
func TeardownEnvironment() {
	if *FlagKeepNS || sharedNS == nil || sharedNsMgr == nil {
		return
	}
	logger := slog.Default()
	sharedNsMgr.DeleteAll(context.Background(), sharedNS)
	logger.Info("test namespaces deleted")
}
