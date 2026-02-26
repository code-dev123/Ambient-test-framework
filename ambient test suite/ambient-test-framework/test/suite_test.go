// test/suite_test.go
package test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestAmbientSuite is the Ginkgo entry point for the root test package.
// Sub-packages (l4, l7, baseline) each have their own suite bootstraps
// that call SetupEnvironment/TeardownEnvironment independently.
func TestAmbientSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ambient Test Suite")
}

var _ = BeforeSuite(func() {
	SetupEnvironment()
})

var _ = AfterSuite(func() {
	TeardownEnvironment()
})
