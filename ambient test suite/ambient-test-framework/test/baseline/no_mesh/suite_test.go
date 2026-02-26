// test/baseline/no_mesh/suite_test.go
package no_mesh

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/test"
)

func TestNoMeshSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "No Mesh Baseline Suite")
}

var _ = BeforeSuite(func() {
	test.SetupEnvironment()
})

var _ = AfterSuite(func() {
	test.TeardownEnvironment()
})
