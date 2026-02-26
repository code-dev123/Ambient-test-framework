// test/l4/l4_authz/suite_test.go
package l4_authz

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/test"
)

func TestL4AuthzSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "L4 Authorization Suite")
}

var _ = BeforeSuite(func() {
	test.SetupEnvironment()
})

var _ = AfterSuite(func() {
	test.TeardownEnvironment()
})
