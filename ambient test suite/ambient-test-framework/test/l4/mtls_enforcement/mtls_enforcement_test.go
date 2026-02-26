// test/l4/mtls_enforcement/mtls_enforcement_test.go
package mtls_enforcement

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

var _ = Describe("mTLS Enforcement: STRICT mode", func() {
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
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancel)

		applier := env.NewApplier()
		values := manifest.TemplateValues{
			Namespace: env.Namespaces.HTTP,
			TestID:    env.RunID,
		}

		result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
		Expect(err).NotTo(HaveOccurred(), "apply peer-auth manifests")
		DeferCleanup(func() {
			cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
			defer c()
			applier.DeleteResult(cleanupCtx, result)
		})

		time.Sleep(5 * time.Second)
	})

	It("mesh client can reach service through ztunnel mTLS", func() {
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		client := &http.Client{Timeout: 5 * time.Second}
		assert.EventuallyHTTPOK(ctx, client, endpoint,
			3*time.Second, 30*time.Second)
	})

	It("ztunnel DaemonSet is running and enforcing mTLS", func() {
		assert.ZTunnelRunning(ctx, env.Clientset)
	})

	It("namespace is enrolled in ambient mesh", func() {
		assert.MTLSEnforced(ctx, env.Clientset, env.Namespaces.HTTP, "podinfo")
	})

	It("ambient label is present and stable (retried with Gomega Eventually)", func() {
		// Use Gomega Eventually instead of the custom pkgwait helper.
		Eventually(func(ctx context.Context) error {
			assert.MTLSEnforced(ctx, env.Clientset, env.Namespaces.HTTP, "podinfo")
			return nil
		}).WithContext(ctx).WithTimeout(20 * time.Second).WithPolling(2 * time.Second).
			Should(Succeed(), "ambient label check failed")
	})
})

var _ = Describe("mTLS Enforcement: PERMISSIVE mode", func() {
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

	It("PERMISSIVE mode allows plaintext connections", func() {
		// No PeerAuthentication applied → default is PERMISSIVE in ambient.
		endpoint := traffic.Endpoint{
			Host: "podinfo." + env.Namespaces.HTTP + ".svc.cluster.local",
			Port: 9898,
			Path: "/healthz",
		}
		client := &http.Client{Timeout: 5 * time.Second}
		assert.EventuallyHTTPOK(ctx, client, endpoint,
			3*time.Second, 30*time.Second)
	})
})
