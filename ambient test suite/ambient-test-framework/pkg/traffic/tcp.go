// pkg/traffic/tcp.go
package traffic

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TCPEndpoint identifies a raw TCP endpoint.
type TCPEndpoint struct {
	Host string
	Port int
}

// Addr returns the dial address.
func (e TCPEndpoint) Addr() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

// TCPConnect attempts a TCP connection to the endpoint.
// Returns nil if the connection succeeds, an error otherwise.
func TCPConnect(ctx context.Context, endpoint TCPEndpoint) error {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint.Addr())
	if err != nil {
		return fmt.Errorf("tcp connect to %s: %w", endpoint.Addr(), err)
	}
	conn.Close()
	return nil
}

// TCPConnectWithRetry retries TCP connection until success or timeout.
func TCPConnectWithRetry(ctx context.Context,
	endpoint TCPEndpoint,
	interval, timeout time.Duration) error {

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := TCPConnect(ctx, endpoint); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return fmt.Errorf("tcp connect to %s timed out after %s",
		endpoint.Addr(), timeout)
}

// TCPExpectRefused asserts that a TCP connection is refused (port blocked).
func TCPExpectRefused(ctx context.Context, endpoint TCPEndpoint) error {
	err := TCPConnect(ctx, endpoint)
	if err == nil {
		return fmt.Errorf("expected connection refused to %s, but it succeeded",
			endpoint.Addr())
	}

	netErr, ok := err.(*net.OpError)
	if !ok {
		return fmt.Errorf("unexpected error type: %w", err)
	}

	if netErr.Op == "dial" {
		return nil // connection refused as expected
	}
	return fmt.Errorf("unexpected tcp error: %w", err)
}
