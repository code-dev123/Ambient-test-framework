// pkg/assert/headers.go
package assert

import (
	. "github.com/onsi/gomega"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// HasHeader asserts that the response contains a header with the expected value.
func HasHeader(resp *traffic.HTTPResponse, header, expected string) {
	Expect(resp).NotTo(BeNil(), "response is nil")
	Expect(resp.Headers.Get(header)).To(Equal(expected),
		"header %q: expected %q", header, expected)
}

// HeaderExists asserts that the response contains the given header (any value).
func HeaderExists(resp *traffic.HTTPResponse, header string) {
	Expect(resp).NotTo(BeNil(), "response is nil")
	Expect(resp.Headers.Get(header)).NotTo(BeEmpty(),
		"expected header %q to be present, but it was absent", header)
}

// HeaderAbsent asserts that the response does NOT contain the given header.
func HeaderAbsent(resp *traffic.HTTPResponse, header string) {
	Expect(resp).NotTo(BeNil(), "response is nil")
	Expect(resp.Headers.Get(header)).To(BeEmpty(),
		"expected header %q to be absent", header)
}

// HasHeaderContaining asserts that a header contains the given substring.
func HasHeaderContaining(resp *traffic.HTTPResponse, header, substring string) {
	Expect(resp).NotTo(BeNil(), "response is nil")
	got := resp.Headers.Get(header)
	Expect(got).NotTo(BeEmpty(), "header %q not present in response", header)
	Expect(got).To(ContainSubstring(substring),
		"header %q: expected to contain %q, got %q", header, substring, got)
}
