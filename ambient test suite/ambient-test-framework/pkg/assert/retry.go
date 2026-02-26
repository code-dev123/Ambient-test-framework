// pkg/assert/retry.go
package assert

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// EventuallyHTTPOK retries an HTTP GET until it returns 200 or the timeout expires.
// Uses Gomega's Eventually for structured polling and failure messages.
func EventuallyHTTPOK(ctx context.Context, client *http.Client,
	endpoint traffic.Endpoint, interval, timeout time.Duration) {

	Eventually(func(ctx context.Context) error {
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		return nil
	}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).
		Should(Succeed(), "endpoint %s never returned HTTP 200 within %s",
			endpoint.URL(), timeout)
}

// EventuallyHTTPStatus retries until the expected HTTP status is returned.
func EventuallyHTTPStatus(ctx context.Context, client *http.Client,
	endpoint traffic.Endpoint, expectedStatus int,
	interval, timeout time.Duration) {

	Eventually(func(ctx context.Context) error {
		resp, err := traffic.HTTPGet(ctx, client, endpoint)
		if err != nil {
			return err
		}
		if resp.StatusCode != expectedStatus {
			return fmt.Errorf("expected status %d, got %d", expectedStatus, resp.StatusCode)
		}
		return nil
	}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).
		Should(Succeed(), "endpoint %s never returned status %d within %s",
			endpoint.URL(), expectedStatus, timeout)
}
