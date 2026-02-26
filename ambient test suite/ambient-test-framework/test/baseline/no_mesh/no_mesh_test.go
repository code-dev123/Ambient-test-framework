// test/baseline/no_mesh/no_mesh_test.go
package no_mesh

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yourorg/ambient-test-framework/pkg/assert"
	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	"github.com/yourorg/ambient-test-framework/test"
)

var _ = Describe("No Mesh Baseline HTTP", func() {
	var (
		env      *test.Environment
		ctx      context.Context
		client   *http.Client
		endpoint traffic.Endpoint
	)

	BeforeEach(func() {
		env = test.GetEnvironment()
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
		DeferCleanup(cancel)

		client = &http.Client{Timeout: 5 * time.Second}
		endpoint = traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.NonMesh + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
	})

	It("plain HTTP GET succeeds in non-mesh namespace", func() {
		assert.EventuallyHTTPOK(ctx, client, endpoint,
			2*time.Second, 30*time.Second)
	})

	It("multiple requests all succeed", func() {
		for i := 0; i < 10; i++ {
			resp, err := traffic.HTTPGet(ctx, client, endpoint)
			assert.NoError(err)
			assert.StatusOK(resp)
		}
	})
})

var _ = Describe("No Mesh Baseline TCP", func() {
	var (
		env      *test.Environment
		ctx      context.Context
		endpoint traffic.TCPEndpoint
	)

	BeforeEach(func() {
		env = test.GetEnvironment()
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
		DeferCleanup(cancel)

		endpoint = traffic.TCPEndpoint{
			Host: "podinfo." + env.Namespaces.NonMesh + ".svc.cluster.local",
			Port: 9898,
		}
	})

	It("TCP connection succeeds", func() {
		err := traffic.TCPConnectWithRetry(ctx, endpoint,
			2*time.Second, 30*time.Second)
		assert.NoError(err)
	})
})

var _ = Describe("No Mesh Isolation", func() {
	var (
		env *test.Environment
		ctx context.Context
	)

	BeforeEach(func() {
		env = test.GetEnvironment()
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
		DeferCleanup(cancel)
	})

	It("non-mesh namespace does not have the ambient label", func() {
		ns, err := env.Clientset.CoreV1().Namespaces().Get(ctx,
			env.Namespaces.NonMesh, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "get namespace %s", env.Namespaces.NonMesh)
		if v, ok := ns.Labels["istio.io/dataplane-mode"]; ok && v == "ambient" {
			Fail("non-mesh namespace " + env.Namespaces.NonMesh +
				" should NOT have ambient label, but does")
		}
	})

	It("ztunnel DaemonSet is running", func() {
		assert.ZTunnelRunning(ctx, env.Clientset)
	})
})
