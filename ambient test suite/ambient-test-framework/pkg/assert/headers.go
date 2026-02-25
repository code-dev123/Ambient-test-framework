// pkg/assert/headers.go
package assert

import (
	"testing"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// HasHeader asserts that the response contains a header with the expected value.
func HasHeader(t *testing.T, resp *traffic.HTTPResponse,
	header, expected string) {

	t.Helper()
	if resp == nil {
		t.Fatal("response is nil")
	}
	got := resp.Headers.Get(header)
	if got != expected {
		t.Errorf("header %q: expected %q, got %q", header, expected, got)
	}
}

// HeaderExists asserts that the response contains the given header (any value).
func HeaderExists(t *testing.T, resp *traffic.HTTPResponse, header string) {
	t.Helper()
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Headers.Get(header) == "" {
		t.Errorf("expected header %q to be present, but it was absent", header)
	}
}

// HeaderAbsent asserts that the response does NOT contain the given header.
func HeaderAbsent(t *testing.T, resp *traffic.HTTPResponse, header string) {
	t.Helper()
	if resp == nil {
		t.Fatal("response is nil")
	}
	if v := resp.Headers.Get(header); v != "" {
		t.Errorf("expected header %q to be absent, but got %q", header, v)
	}
}

// HasHeaderContaining asserts that a header contains the given substring.
func HasHeaderContaining(t *testing.T, resp *traffic.HTTPResponse,
	header, substring string) {

	t.Helper()
	if resp == nil {
		t.Fatal("response is nil")
	}
	got := resp.Headers.Get(header)
	if got == "" {
		t.Errorf("header %q not present in response", header)
		return
	}
	// Simple substring check using standard library
	if len(got) < len(substring) {
		t.Errorf("header %q: expected to contain %q, got %q",
			header, substring, got)
		return
	}
	for i := 0; i <= len(got)-len(substring); i++ {
		if got[i:i+len(substring)] == substring {
			return
		}
	}
	t.Errorf("header %q: expected to contain %q, got %q",
		header, substring, got)
}
