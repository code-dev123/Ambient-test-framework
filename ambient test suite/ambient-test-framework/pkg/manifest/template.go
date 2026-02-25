// pkg/manifest/template.go
package manifest

import (
	"bytes"
	"text/template"
)

// TemplateValues are injected into manifest YAML at apply-time.
// This lets the same manifest work across test runs with
// different namespaces, versions, and labels.
type TemplateValues struct {
	Namespace        string
	TestID           string // unique per test run, e.g., "a1b2c3"
	PodInfoImageV1   string
	PodInfoImageV2   string
	CanaryWeight     int
	StableWeight     int
	ServiceName      string
	GatewayClassName string // e.g., "istio" for ambient waypoint
	// Add more as needed — users can extend with custom fields
	Extra map[string]string
}

// RenderTemplate renders a Go template string with the provided values.
func RenderTemplate(raw string, values TemplateValues) (string, error) {
	tmpl, err := template.New("manifest").
		Option("missingkey=error"). // Fail fast on typos
		Parse(raw)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}
