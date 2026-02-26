// test/l4/cross_namespace/suite_test.go
package cross_namespace

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/test"
)

func TestCrossNamespaceSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "L4 Cross-Namespace Suite")
}

var _ = BeforeSuite(func() {
	test.SetupEnvironment()
})

var _ = AfterSuite(func() {
	test.TeardownEnvironment()
})
