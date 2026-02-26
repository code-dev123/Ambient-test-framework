// test/setup_test.go
package test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// HTTPEndpointInNamespace returns a traffic.Endpoint for a named service
// in the HTTP test namespace.
func HTTPEndpointInNamespace(env *Environment,
	serviceName, path string, port int) traffic.Endpoint {

	return traffic.Endpoint{
		Host: fmt.Sprintf("%s.%s.svc.cluster.local",
			serviceName, env.Namespaces.HTTP),
		Port: port,
		Path: path,
	}
}

// GRPCEndpointInNamespace returns a GRPCEndpoint for a service in the
// GRPC test namespace.
func GRPCEndpointInNamespace(env *Environment,
	serviceName string, port int) traffic.GRPCEndpoint {

	return traffic.GRPCEndpoint{
		Host: fmt.Sprintf("%s.%s.svc.cluster.local",
			serviceName, env.Namespaces.GRPC),
		Port: port,
	}
}

// WaitForHTTPService polls until the named service returns HTTP 200.
func WaitForHTTPService(ctx context.Context,
	env *Environment, serviceName, path string, port int,
	timeout time.Duration) {

	endpoint := HTTPEndpointInNamespace(env, serviceName, path, port)
	client := &http.Client{Timeout: 5 * time.Second}

	Eventually(func(ctx context.Context) error {
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}).WithContext(ctx).WithTimeout(timeout).WithPolling(3 * time.Second).
		Should(Succeed(), "service %s never became ready", serviceName)
}
