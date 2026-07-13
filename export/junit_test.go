package export

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jstemmer/go-junit-report/v2/junit"
	"github.com/mtracer-project/mtracer/test"
)

func TestJunitExporter_Export(t *testing.T) {
	tempDir := t.TempDir()
	exporter := newJunitExporter(tempDir, "test.xml", time.Now())

	results := []*test.TestSuite{
		test.NewTestSuite("test 1", []*test.TestResult{
			{
				Passed:   true,
				Duration: 5 * time.Second,
				Args:     []any{"key1", "value1"},
			},
		}),
		test.NewTestSuite("test 2", []*test.TestResult{
			{
				Passed:   false,
				Duration: 2 * time.Second,
				Args:     []any{"error", "some error"},
			},
		}),
	}

	err := exporter.Export(results)
	if err != nil {
		t.Fatalf("unexpected error exporting: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "test.xml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file %s to be created", expectedPath)
	}

	data, err := os.ReadFile(expectedPath) // nolint:gosec
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}

	var suites junit.Testsuites
	err = xml.Unmarshal(data, &suites)
	if err != nil {
		t.Fatalf("unexpected error unmarshaling xml: %v", err)
	}

	if len(suites.Suites) != 2 {
		t.Fatalf("expected 2 testsuites, got %d", len(suites.Suites))
	}

	suite := suites.Suites[0]
	if suite.Tests != 1 {
		t.Errorf("expected 1 test, got %d", suite.Tests)
	}
	if suite.Failures != 0 {
		t.Errorf("expected 0 failure, got %d", suite.Failures)
	}
	if suite.Time != "5.000" {
		t.Errorf("expected total time 5.000, got %s", suite.Time)
	}

	if len(suite.Testcases) != 1 {
		t.Fatalf("expected 1 testcase, got %d", len(suite.Testcases))
	}

	if suite.Testcases[0].Name != "test 1 (Run 1)" || suite.Testcases[0].Failure != nil {
		t.Errorf("expected test 1 to pass")
	}
	if suite.Testcases[0].Time != "5.000" {
		t.Errorf("expected test 1 time 5.000, got %s", suite.Testcases[0].Time)
	}

	suite2 := suites.Suites[1]
	if suite2.Testcases[0].Name != "test 2 (Run 1)" || suite2.Testcases[0].Failure == nil {
		t.Errorf("expected test 2 to fail")
	}
	if suite2.Testcases[0].Time != "2.000" {
		t.Errorf("expected test 2 time 2.000, got %s", suite2.Testcases[0].Time)
	}
}

func TestJunitExporter_Format(t *testing.T) {
	exporter := newJunitExporter("dir", "file", time.Now())
	if exporter.Format() != JUNIT_FORMAT {
		t.Errorf("expected format %s, got %s", JUNIT_FORMAT, exporter.Format())
	}
}
