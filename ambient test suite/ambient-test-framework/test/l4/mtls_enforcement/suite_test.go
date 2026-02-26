// test/l4/mtls_enforcement/suite_test.go
package mtls_enforcement

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/test"
)

func TestMTLSEnforcementSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "L4 mTLS Enforcement Suite")
}

var _ = BeforeSuite(func() {
	test.SetupEnvironment()
})

var _ = AfterSuite(func() {
	test.TeardownEnvironment()
})
