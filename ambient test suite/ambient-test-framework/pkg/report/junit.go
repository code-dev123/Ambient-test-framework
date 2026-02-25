// pkg/report/junit.go
package report

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

// JUnitTestSuites is the root element of a JUnit XML report.
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr,omitempty"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Time       float64          `xml:"time,attr"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a single test suite in JUnit XML format.
type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Name       string          `xml:"name,attr"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Errors     int             `xml:"errors,attr"`
	Skipped    int             `xml:"skipped,attr"`
	Time       float64         `xml:"time,attr"`
	Timestamp  string          `xml:"timestamp,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase `xml:"testcase"`
}

// JUnitProperty is a key-value property in a JUnit XML report.
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitTestCase represents a single test case in JUnit XML format.
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Classname string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
}

// JUnitFailure represents a test failure.
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// JUnitError represents a test error.
type JUnitError struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// JUnitSkipped represents a skipped test.
type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// WriteJUnit writes a JUnit XML report to the given file path.
func WriteJUnit(path string, suites *JUnitTestSuites) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create junit file: %w", err)
	}
	defer f.Close()

	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if _, err := fmt.Fprint(f, xml.Header); err != nil {
		return err
	}
	return enc.Encode(suites)
}

// NewTestSuite creates a JUnitTestSuite with the current timestamp.
func NewTestSuite(name string) *JUnitTestSuite {
	return &JUnitTestSuite{
		Name:      name,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// AddTestCase appends a test case to the suite and updates counters.
func (s *JUnitTestSuite) AddTestCase(tc JUnitTestCase) {
	s.Tests++
	s.Time += tc.Time
	if tc.Failure != nil {
		s.Failures++
	}
	if tc.Error != nil {
		s.Errors++
	}
	if tc.Skipped != nil {
		s.Skipped++
	}
	s.TestCases = append(s.TestCases, tc)
}
