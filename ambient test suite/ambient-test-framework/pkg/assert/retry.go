// pkg/assert/retry.go
package assert

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// EventuallyHTTPOK retries an HTTP GET until it returns 200 or the context expires.
func EventuallyHTTPOK(ctx context.Context, t *testing.T,
	client *http.Client, endpoint traffic.Endpoint,
	interval, timeout time.Duration) {

	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	var lastStatus int

	for time.Now().Before(deadline) {
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		if err == nil && resp.StatusCode == http.StatusOK {
			return
		}
		if err != nil {
			lastErr = err
		} else {
			lastStatus = resp.StatusCode
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled waiting for HTTP OK on %s", endpoint.URL())
		case <-time.After(interval):
		}
	}

	if lastErr != nil {
		t.Fatalf("endpoint %s never returned HTTP 200 within %s: last error: %v",
			endpoint.URL(), timeout, lastErr)
	} else {
		t.Fatalf("endpoint %s never returned HTTP 200 within %s: last status: %d",
			endpoint.URL(), timeout, lastStatus)
	}
}

// EventuallyHTTPStatus retries until the expected HTTP status is returned.
func EventuallyHTTPStatus(ctx context.Context, t *testing.T,
	client *http.Client, endpoint traffic.Endpoint,
	expectedStatus int,
	interval, timeout time.Duration) {

	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		if err == nil && resp.StatusCode == expectedStatus {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled waiting for status %d on %s",
				expectedStatus, endpoint.URL())
		case <-time.After(interval):
		}
	}

	t.Fatalf("endpoint %s never returned status %d within %s",
		endpoint.URL(), expectedStatus, timeout)
}

// Eventually runs fn until it returns true or timeout elapses.
func Eventually(t *testing.T, fn func() bool,
	interval, timeout time.Duration, msgFmt string, args ...interface{}) {

	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(interval)
	}

	msg := msgFmt
	if len(args) > 0 {
		// Simple format without fmt dependency cycle issues
		t.Helper()
	}
	_ = msg
	t.Errorf("condition not met within %s: "+msgFmt, append([]interface{}{timeout}, args...)...)
}
