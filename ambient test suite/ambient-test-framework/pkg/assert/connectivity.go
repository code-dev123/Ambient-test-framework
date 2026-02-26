// pkg/assert/connectivity.go
package assert

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// StatusOK asserts that an HTTP response has status 200.
func StatusOK(resp *traffic.HTTPResponse) {
	Expect(resp).NotTo(BeNil(), "response is nil")
	Expect(resp.StatusCode).To(Equal(http.StatusOK),
		"expected status 200, got %d", resp.StatusCode)
}

// StatusCode asserts that an HTTP response has the expected status code.
func StatusCode(resp *traffic.HTTPResponse, expected int) {
	Expect(resp).NotTo(BeNil(), "response is nil")
	Expect(resp.StatusCode).To(Equal(expected),
		"expected status %d, got %d", expected, resp.StatusCode)
}

// StatusForbidden asserts that an HTTP response has status 403.
func StatusForbidden(resp *traffic.HTTPResponse) {
	StatusCode(resp, http.StatusForbidden)
}

// NoError asserts that err is nil.
func NoError(err error) {
	Expect(err).NotTo(HaveOccurred())
}

// Error asserts that err is not nil.
func Error(err error) {
	Expect(err).To(HaveOccurred())
}

// HTTPReachable asserts that an endpoint responds with HTTP 200.
func HTTPReachable(ctx context.Context, client *http.Client, endpoint traffic.Endpoint) {
	resp, err := traffic.HTTPGet(ctx, client, endpoint)
	NoError(err)
	StatusOK(resp)
}

// HTTPUnreachable asserts that an endpoint returns HTTP 403/503 or the connection fails.
func HTTPUnreachable(ctx context.Context, client *http.Client, endpoint traffic.Endpoint) {
	resp, err := traffic.HTTPGet(ctx, client, endpoint)
	if err != nil {
		// Connection refused / reset — counts as unreachable
		return
	}
	Expect(resp.StatusCode).To(SatisfyAny(
		Equal(http.StatusForbidden),
		Equal(http.StatusServiceUnavailable),
	), "expected endpoint to be unreachable, got status %d", resp.StatusCode)
}

// InDelta asserts that got is within delta of expected.
func InDelta(expected, got, delta float64, msgFmt string, args ...any) {
	msg := fmt.Sprintf(msgFmt, args...)
	Expect(got).To(BeNumerically("~", expected, delta), msg)
}
