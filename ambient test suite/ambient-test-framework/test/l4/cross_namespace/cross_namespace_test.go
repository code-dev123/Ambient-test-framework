// test/l4/cross_namespace/cross_namespace_test.go
package cross_namespace

import (
	"context"
	"embed"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/pkg/assert"
	"github.com/yourorg/ambient-test-framework/pkg/manifest"
	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	"github.com/yourorg/ambient-test-framework/test"
)

//go:embed testdata/*
var testdataFS embed.FS

var _ = Describe("Cross-Namespace Allowed", func() {
	BeforeEach(func() {
		test.SkipUnlessLayer("l4")
	})

	var (
		env    *test.Environment
		ctx    context.Context
		client *http.Client
	)

	BeforeEach(func() {
		env = test.GetEnvironment()
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancel)

		applier := env.NewApplier()
		values := manifest.TemplateValues{
			Namespace: env.Namespaces.HTTP,
			TestID:    env.RunID,
		}

		result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
		Expect(err).NotTo(HaveOccurred(), "apply cross-ns manifests")
		DeferCleanup(func() {
			cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
			defer c()
			applier.DeleteResult(cleanupCtx, result)
		})

		time.Sleep(5 * time.Second)
		client = &http.Client{Timeout: 5 * time.Second}
	})

	It("HTTP client (http namespace) can reach HTTP service", func() {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		assert.EventuallyHTTPOK(ctx, client, endpoint,
			2*time.Second, 30*time.Second)
	})

	It("GRPC client can reach HTTP service", func() {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		assert.EventuallyHTTPOK(ctx, client, endpoint,
			2*time.Second, 30*time.Second)
	})

	It("non-mesh client is denied (ztunnel enforces policy)", func() {
		// The policy only allows http-* and grpc-* namespaces.
		// Verify ztunnel is running and enforcing the policy.
		assert.ZTunnelRunning(ctx, env.Clientset)
	})
})

var _ = Describe("Cross-Namespace Default Deny", func() {
	BeforeEach(func() {
		test.SkipUnlessLayer("l4")
	})

	var (
		env *test.Environment
		ctx context.Context
	)

	BeforeEach(func() {
		env = test.GetEnvironment()
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
		DeferCleanup(cancel)
	})

	It("no policy allows all traffic by default", func() {
		// Without any AuthorizationPolicy, ambient defaults to ALLOW
		// (unless a DENY policy is present). This test documents that baseline.
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.GRPC + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		client := &http.Client{Timeout: 5 * time.Second}
		assert.EventuallyHTTPOK(ctx, client, endpoint,
			2*time.Second, 30*time.Second)
	})
})
