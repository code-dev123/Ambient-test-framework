// test/setup_test.go
package test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
	pkgwait "github.com/yourorg/ambient-test-framework/pkg/wait"
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
func WaitForHTTPService(ctx context.Context, t *testing.T,
	env *Environment, serviceName, path string, port int,
	timeout time.Duration) {

	t.Helper()
	endpoint := HTTPEndpointInNamespace(env, serviceName, path, port)
	client := &http.Client{Timeout: 5 * time.Second}

	err := pkgwait.UntilNoError(ctx,
		func(ctx context.Context) error {
			resp, err := traffic.HTTPGet(ctx, client, endpoint)
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("status %d", resp.StatusCode)
			}
			return nil
		},
		3*time.Second, timeout)

	if err != nil {
		t.Fatalf("service %s never became ready: %v", serviceName, err)
	}
}
