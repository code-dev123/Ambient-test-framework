// pkg/report/json.go
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// TestRunReport is the top-level JSON report for a test run.
type TestRunReport struct {
	RunID     string        `json:"run_id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  string        `json:"duration"`
	Passed    int           `json:"passed"`
	Failed    int           `json:"failed"`
	Skipped   int           `json:"skipped"`
	Total     int           `json:"total"`
	Results   []TestResult  `json:"results"`
	Metadata  RunMetadata   `json:"metadata"`
}

// TestResult holds the result of a single test.
type TestResult struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"` // "passed", "failed", "skipped"
	Duration  time.Duration `json:"duration_ns"`
	Error     string        `json:"error,omitempty"`
	SubTests  []TestResult  `json:"sub_tests,omitempty"`
}

// RunMetadata holds environment metadata for the test run.
type RunMetadata struct {
	IstioVersion string `json:"istio_version,omitempty"`
	K8sVersion   string `json:"k8s_version,omitempty"`
	ClusterName  string `json:"cluster_name,omitempty"`
	Region       string `json:"region,omitempty"`
	Layer        string `json:"layer,omitempty"` // "l4", "l7", "all"
}

// WriteJSON writes a TestRunReport to the given file path as JSON.
func WriteJSON(path string, report *TestRunReport) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// NewTestRunReport creates a TestRunReport with the given run ID.
func NewTestRunReport(runID string, meta RunMetadata) *TestRunReport {
	return &TestRunReport{
		RunID:     runID,
		StartTime: time.Now(),
		Metadata:  meta,
	}
}

// Finalize closes the report and calculates totals.
func (r *TestRunReport) Finalize() {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime).String()
	r.Total = r.Passed + r.Failed + r.Skipped
}

// AddResult appends a test result and updates counters.
func (r *TestRunReport) AddResult(result TestResult) {
	switch result.Status {
	case "passed":
		r.Passed++
	case "failed":
		r.Failed++
	case "skipped":
		r.Skipped++
	}
	r.Results = append(r.Results, result)
}
