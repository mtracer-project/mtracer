package export

import (
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/test"
	testutils "github.com/mtracer-project/mtracer/testUtils"
)

func TestStdoutExporter_Export(t *testing.T) {
	exporter := newStdoutExporter(1)

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

	output := testutils.CaptureStdout(t, func() {
		err := exporter.Export(results)
		if err != nil {
			t.Fatalf("unexpected error exporting: %v", err)
		}
	})

	if !strings.Contains(output, "TEST NAME") || !strings.Contains(output, "STATUS") || !strings.Contains(output, "DETAILS") {
		t.Errorf("expected table headers, got %s", output)
	}

	if !strings.Contains(output, "test 1") || !strings.Contains(output, "PASSED") {
		t.Errorf("expected test 1 to be PASSED")
	}

	if !strings.Contains(output, "test 2") || !strings.Contains(output, "FAILED") {
		t.Errorf("expected test 2 to be FAILED")
	}
}

func TestStdoutExporter_ExportMultipleRuns(t *testing.T) {
	exporter := newStdoutExporter(2)

	results := []*test.TestSuite{
		test.NewTestSuite("test 1", []*test.TestResult{
			{
				Passed: true,
				Args:   []any{"key1", "value1"},
			},
			{
				Passed: false,
				Args:   []any{"error", "some error"},
			},
		}),
	}

	output := testutils.CaptureStdout(t, func() {
		err := exporter.Export(results)
		if err != nil {
			t.Fatalf("unexpected error exporting: %v", err)
		}
	})

	if !strings.Contains(output, "Test: test 1") || !strings.Contains(output, "RUN") || !strings.Contains(output, "STATUS") || !strings.Contains(output, "DETAILS") {
		t.Errorf("expected table headers, got %s", output)
	}

	if !strings.Contains(output, "1") || !strings.Contains(output, "PASSED") {
		t.Errorf("expected test 1 to be PASSED")
	}

	if !strings.Contains(output, "2") || !strings.Contains(output, "FAILED") {
		t.Errorf("expected test 2 to be FAILED")
	}
}

func TestStdoutExporter_ExportEmpty(t *testing.T) {
	exporter := newStdoutExporter(1)

	output := testutils.CaptureStdout(t, func() {
		err := exporter.Export(nil)
		if err != nil {
			t.Fatalf("unexpected error exporting: %v", err)
		}
	})

	if output != "" {
		t.Errorf("expected empty output, got %q", output)
	}
}

func TestStdoutExporter_Format(t *testing.T) {
	exporter := newStdoutExporter(1)
	if exporter.Format() != STDOUT_FORMAT {
		t.Errorf("expected format %s, got %s", STDOUT_FORMAT, exporter.Format())
	}
}

func TestDisplayTestsSummary(t *testing.T) {
	results := []*test.TestSuite{
		test.NewTestSuite("test 1", []*test.TestResult{
			{
				Passed: true,
				Args:   []any{"key1", "value1"},
			},
		}),
	}

	output := testutils.CaptureStdout(t, func() {
		err := DisplayTestsSummary(results, "OK", "KO")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "test 1") || !strings.Contains(output, "OK") {
		t.Errorf("expected custom OK status")
	}
}
