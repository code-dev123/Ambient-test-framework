// cmd/ambient-test/main.go
// Optional Cobra CLI runner — wraps `go test` with a friendlier UX.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagLayer      string
	flagKubeconfig string
	flagEKSCluster string
	flagRegion     string
	flagKeepNS     bool
	flagVerbose    bool
	flagTimeout    string
	flagRun        string
	flagReport     string
)

func main() {
	root := &cobra.Command{
		Use:   "ambient-test",
		Short: "Istio Ambient mesh test runner",
		Long: `ambient-test is a CLI wrapper around 'go test' for the
Istio Ambient test suite. It passes all flags through to the
underlying go test invocation.`,
		RunE: runTests,
	}

	root.Flags().StringVar(&flagLayer, "layer", "all",
		"Test layer to run: l4, l7, or all")
	root.Flags().StringVar(&flagKubeconfig, "kubeconfig", "",
		"Path to kubeconfig file")
	root.Flags().StringVar(&flagEKSCluster, "eks-cluster", "",
		"EKS cluster name (optional, used for auto kubeconfig generation)")
	root.Flags().StringVar(&flagRegion, "region", "us-west-2",
		"AWS region")
	root.Flags().BoolVar(&flagKeepNS, "keep-ns", false,
		"Preserve test namespaces after run")
	root.Flags().BoolVarP(&flagVerbose, "verbose", "v", false,
		"Verbose test output")
	root.Flags().StringVar(&flagTimeout, "timeout", "30m",
		"Overall test timeout (go duration format)")
	root.Flags().StringVar(&flagRun, "run", "",
		"Run only tests matching this regex (passed to go test -run)")
	root.Flags().StringVar(&flagReport, "report", "",
		"Path to write JUnit XML report")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTests(cmd *cobra.Command, args []string) error {
	// Determine test package path based on layer
	var pkg string
	switch flagLayer {
	case "l4":
		pkg = "./test/l4/..."
	case "l7":
		pkg = "./test/l7/..."
	case "all":
		pkg = "./test/..."
	default:
		return fmt.Errorf("unknown layer %q: must be l4, l7, or all", flagLayer)
	}

	// Build go test arguments
	goArgs := []string{"test", pkg, "-timeout", flagTimeout}

	if flagVerbose {
		goArgs = append(goArgs, "-v")
	}
	if flagRun != "" {
		goArgs = append(goArgs, "-run", flagRun)
	}

	// Pass our custom flags to the test binary
	testFlags := buildTestFlags()
	if len(testFlags) > 0 {
		goArgs = append(goArgs, testFlags...)
	}

	fmt.Printf("Running: go %s\n", strings.Join(goArgs, " "))

	c := exec.Command("go", goArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()

	return c.Run()
}

func buildTestFlags() []string {
	var flags []string

	if flagLayer != "" {
		flags = append(flags, "-layer", flagLayer)
	}
	if flagKubeconfig != "" {
		flags = append(flags, "-kubeconfig", flagKubeconfig)
	}
	if flagEKSCluster != "" {
		flags = append(flags, "-eks-cluster", flagEKSCluster)
	}
	if flagRegion != "" {
		flags = append(flags, "-region", flagRegion)
	}
	if flagKeepNS {
		flags = append(flags, "-keep-ns")
	}

	return flags
}
