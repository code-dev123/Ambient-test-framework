// pkg/assert/connectivity.go
package assert

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// StatusOK asserts that an HTTP response has status 200.
func StatusOK(t *testing.T, resp *traffic.HTTPResponse) {
	t.Helper()
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// StatusCode asserts that an HTTP response has the expected status code.
func StatusCode(t *testing.T, resp *traffic.HTTPResponse, expected int) {
	t.Helper()
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.StatusCode != expected {
		t.Errorf("expected status %d, got %d", expected, resp.StatusCode)
	}
}

// StatusForbidden asserts that an HTTP response has status 403.
func StatusForbidden(t *testing.T, resp *traffic.HTTPResponse) {
	t.Helper()
	StatusCode(t, resp, http.StatusForbidden)
}

// NoError asserts that err is nil.
func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Error asserts that err is not nil.
func Error(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// HTTPReachable asserts that an endpoint responds with HTTP 200.
func HTTPReachable(ctx context.Context, t *testing.T,
	client *http.Client, endpoint traffic.Endpoint) {

	t.Helper()
	resp, err := traffic.HTTPGet(ctx, client, endpoint)
	NoError(t, err)
	StatusOK(t, resp)
}

// HTTPUnreachable asserts that an endpoint returns HTTP 403 or connection fails.
func HTTPUnreachable(ctx context.Context, t *testing.T,
	client *http.Client, endpoint traffic.Endpoint) {

	t.Helper()
	resp, err := traffic.HTTPGet(ctx, client, endpoint)
	if err != nil {
		// Connection refused / reset — also counts as unreachable
		return
	}
	if resp.StatusCode == http.StatusForbidden ||
		resp.StatusCode == http.StatusServiceUnavailable {
		return
	}
	t.Errorf("expected endpoint to be unreachable, got status %d",
		resp.StatusCode)
}

// InDelta asserts that got is within delta of expected.
func InDelta(t *testing.T, expected, got, delta float64, msgFmt string, args ...interface{}) {
	t.Helper()
	diff := expected - got
	if diff < 0 {
		diff = -diff
	}
	if diff > delta {
		msg := fmt.Sprintf(msgFmt, args...)
		t.Errorf("%s (diff=%.2f, tolerance=%.2f)", msg, diff, delta)
	}
}
