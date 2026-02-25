// test/framework.go
package test

import (
	"flag"
	"log/slog"
	"sync"
	"testing"

	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"

	"github.com/yourorg/ambient-test-framework/pkg/config"
	"github.com/yourorg/ambient-test-framework/pkg/manifest"
	"github.com/yourorg/ambient-test-framework/pkg/namespace"
	"github.com/yourorg/ambient-test-framework/pkg/workload"
)

// LayerFlag is the -layer flag value, registered here so sub-packages
// can call SkipUnlessLayer without importing a test-only file.
var LayerFlag = flag.String("layer", "all", "l4, l7, all")

// SkipUnlessLayer skips the test if -layer does not match the given layer.
// Sub-test packages import this from the test package (framework.go, not a _test.go).
func SkipUnlessLayer(t *testing.T, layer string) {
	t.Helper()
	if *LayerFlag != "all" && *LayerFlag != layer {
		t.Skipf("skipping: -layer=%s (this test is layer=%s)", *LayerFlag, layer)
	}
}

// Environment is the shared state created once in TestMain.
type Environment struct {
	Clientset       kubernetes.Interface
	DynClient       dynamic.Interface
	DiscoveryClient kubernetes.Interface
	Namespaces      namespace.NamespaceSet
	Workloads       workload.WorkloadSet
	Config          *config.TestConfig
	RunID           string // unique per test run, e.g., "a1b2c3"
	Logger          *slog.Logger
}

var (
	globalEnv *Environment
	envOnce   sync.Once
)

// SetEnvironment is called by TestMain after setup.
func SetEnvironment(env *Environment) {
	globalEnv = env
}

// GetEnvironment retrieves the shared environment.
// Fails the test if TestMain hasn't set it up.
func GetEnvironment(t *testing.T) *Environment {
	t.Helper()
	if globalEnv == nil {
		t.Fatal("test environment not initialized — is TestMain running?")
	}
	return globalEnv
}

// NewApplier creates a manifest.Applier tied to this environment.
func (e *Environment) NewApplier() *manifest.Applier {
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(
		memory.NewMemCacheClient(
			e.DiscoveryClient.Discovery()))
	return manifest.NewApplier(e.DynClient, mapper, e.Logger)
}

// LoadConfig loads the default config file, falling back to defaults on error.
func LoadConfig() *config.TestConfig {
	cfg, err := config.LoadFile("../config/default.yaml")
	if err != nil {
		return config.Defaults()
	}
	return cfg
}
