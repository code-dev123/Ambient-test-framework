// pkg/traffic/grpc.go
package traffic

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCEndpoint identifies a gRPC service endpoint.
type GRPCEndpoint struct {
	Host    string
	Port    int
	Service string // for health checks
}

// Addr returns the dial address.
func (e GRPCEndpoint) Addr() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

// GRPCHealthCheck dials the endpoint and calls the gRPC health check service.
func GRPCHealthCheck(ctx context.Context,
	endpoint GRPCEndpoint) error {

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	//nolint:staticcheck // DialContext is the correct API for grpc v1.60
	conn, err := grpc.DialContext(dialCtx,
		endpoint.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("dial %s: %w", endpoint.Addr(), err)
	}
	defer conn.Close()

	state := conn.GetState()
	if state != connectivity.Ready {
		return fmt.Errorf("connection to %s not ready (state=%s)",
			endpoint.Addr(), state)
	}

	hc := grpc_health_v1.NewHealthClient(conn)
	resp, err := hc.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: endpoint.Service,
	})
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("service not serving: status=%s", resp.Status)
	}
	return nil
}

// GRPCDial creates a gRPC connection to the endpoint.
func GRPCDial(ctx context.Context,
	endpoint GRPCEndpoint) (*grpc.ClientConn, error) {

	//nolint:staticcheck // DialContext is the correct API for grpc v1.60
	conn, err := grpc.DialContext(ctx,
		endpoint.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", endpoint.Addr(), err)
	}
	return conn, nil
}
