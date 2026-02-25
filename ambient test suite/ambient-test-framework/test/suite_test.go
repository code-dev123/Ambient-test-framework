// test/suite_test.go
package test

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/yourorg/ambient-test-framework/pkg/cluster"
	"github.com/yourorg/ambient-test-framework/pkg/id"
	"github.com/yourorg/ambient-test-framework/pkg/namespace"
	"github.com/yourorg/ambient-test-framework/pkg/workload"
	"testing"
)

var (
	// flagLayer is declared in framework.go as LayerFlag.
	// Additional TestMain-only flags:
	flagKeepNS     = flag.Bool("keep-ns", false, "preserve namespaces after test run")
	flagKubeconfig = flag.String("kubeconfig", "", "kubeconfig path")
	flagEKSCluster = flag.String("eks-cluster", "", "EKS cluster name")
	flagRegion     = flag.String("region", "us-west-2", "AWS region")
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx := context.Background()
	runID := id.Short() // e.g., "a1b2c3"

	logger := slog.Default()

	// 1. Connect to cluster
	provider := cluster.NewEKSProvider()
	clientset, dynClient, err := provider.Connect(ctx, cluster.ConnectOpts{
		Kubeconfig:  *flagKubeconfig,
		ClusterName: *flagEKSCluster,
		Region:      *flagRegion,
	})
	if err != nil {
		logger.Error("cluster connect failed", "error", err)
		os.Exit(1)
	}

	// 2. Pre-flight healthcheck
	status, err := provider.Healthcheck(ctx)
	if err != nil {
		logger.Error("pre-flight healthcheck failed", "error", err)
		os.Exit(1)
	}
	if !status.AmbientEnabled {
		logger.Error("ambient mesh not enabled on cluster — ztunnel DaemonSet not found")
		os.Exit(1)
	}
	logger.Info("cluster connected",
		"nodes", status.NodeCount,
		"ambient", status.AmbientEnabled,
		"waypoint", status.WaypointInstalled,
		"run_id", runID)

	// 3. Create test namespaces (shared across all scenarios)
	nsMgr := namespace.NewManager(clientset)
	ns, err := nsMgr.CreateTestNamespaces(ctx, namespace.Config{
		HTTPPrefix:    "http",
		GRPCPrefix:    "grpc",
		NonMeshPrefix: "non-mesh",
		Suffix:        runID,
		AmbientLabel:  true,
	})
	if err != nil {
		logger.Error("namespace creation failed", "error", err)
		os.Exit(1)
	}
	logger.Info("test namespaces created",
		"http", ns.HTTP,
		"grpc", ns.GRPC,
		"non-mesh", ns.NonMesh)

	// 4. Deploy baseline client workloads (traffic sources)
	deployer := workload.NewPodInfoDeployer(clientset)
	wl, err := deployer.DeployClients(ctx, ns)
	if err != nil {
		logger.Error("workload deploy failed", "error", err)
		os.Exit(1)
	}

	// 5. Set the shared environment for all test packages
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

	// 6. Run all tests
	code := m.Run()

	// 7. Teardown — delete namespaces unless --keep-ns
	if !*flagKeepNS {
		nsMgr.DeleteAll(context.Background(), ns)
		logger.Info("test namespaces deleted")
	} else {
		logger.Info("--keep-ns set; namespaces preserved",
			"http", ns.HTTP, "grpc", ns.GRPC)
	}

	os.Exit(code)
}
