// test/framework.go
package test

import (
	"flag"
	"fmt"
	"log/slog"
	"sync"

	ginkgo "github.com/onsi/ginkgo/v2"
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
var LayerFlag = flag.String("layer", "all", "l4, all")

// SkipUnlessLayer skips the current Ginkgo spec if -layer does not match.
func SkipUnlessLayer(layer string) {
	if *LayerFlag != "all" && *LayerFlag != layer {
		ginkgo.Skip(fmt.Sprintf("skipping: -layer=%s (this test is layer=%s)", *LayerFlag, layer))
	}
}

// Environment is the shared state created once in BeforeSuite.
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
	envMu     sync.Mutex
)

// SetEnvironment is called by BeforeSuite after setup.
func SetEnvironment(env *Environment) {
	envMu.Lock()
	defer envMu.Unlock()
	globalEnv = env
}

// GetEnvironment retrieves the shared environment.
// Fails the current Ginkgo spec if BeforeSuite hasn't set it up.
func GetEnvironment() *Environment {
	envMu.Lock()
	defer envMu.Unlock()
	if globalEnv == nil {
		ginkgo.Fail("test environment not initialized — is BeforeSuite running?")
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
