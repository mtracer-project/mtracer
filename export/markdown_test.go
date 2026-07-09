package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mtrace-project/mtrace/test"
)

func TestMarkdownExporter_Export(t *testing.T) {
	tempDir := t.TempDir()
	exporter := newMarkdownExporter(1, tempDir, "test.md", time.Now())

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

	expectedPath := filepath.Join(tempDir, "test.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file %s to be created", expectedPath)
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Test Results Report") {
		t.Errorf("expected header in markdown")
	}
	if !strings.Contains(content, "* **Passed:** 1") {
		t.Errorf("expected 1 passed test in summary")
	}
	if !strings.Contains(content, "* **Failed:** 1") {
		t.Errorf("expected 1 failed test in summary")
	}
	if !strings.Contains(content, "✅ Passed") || !strings.Contains(content, "test 1") {
		t.Errorf("expected test 1 to be marked as passed")
	}
	if !strings.Contains(content, "❌ Failed") || !strings.Contains(content, "test 2") {
		t.Errorf("expected test 2 to be marked as failed")
	}
	if !strings.Contains(content, "- key1: value1") {
		t.Errorf("expected details to be included")
	}
	if !strings.Contains(content, "* **Total Duration:** 7.000 seconds") {
		t.Errorf("expected total duration 7.000s to be in summary")
	}
	if !strings.Contains(content, "* **Duration:** 5.000 seconds") {
		t.Errorf("expected test 1 duration 5.000s to be in detailed results")
	}
	if !strings.Contains(content, "* **Duration:** 2.000 seconds") {
		t.Errorf("expected test 2 duration 2.000s to be in detailed results")
	}
}

func TestMarkdownExporter_Format(t *testing.T) {
	exporter := newMarkdownExporter(1, "dir", "file", time.Now())
	if exporter.Format() != MARKDOWN_FORMAT {
		t.Errorf("expected format %s, got %s", MARKDOWN_FORMAT, exporter.Format())
	}
}

func TestMarkdownExporter_ExportMultipleRuns(t *testing.T) {
	tempDir := t.TempDir()
	exporter := newMarkdownExporter(2, tempDir, "test_multiple.md", time.Now())

	results := []*test.TestSuite{
		test.NewTestSuite("test 1", []*test.TestResult{
			{
				Passed:   true,
				Duration: 5 * time.Second,
				Args:     []any{"key1", "value1"},
			},
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

	expectedPath := filepath.Join(tempDir, "test_multiple.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file %s to be created", expectedPath)
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# Test Results Report") {
		t.Errorf("expected header in markdown")
	}
	if !strings.Contains(content, "* **Total Tests:** 1") {
		t.Errorf("expected 1 total tests in summary")
	}
	if !strings.Contains(content, "* **Total Runs:** 2") {
		t.Errorf("expected 2 total runs in summary")
	}
	if !strings.Contains(content, "* **Passed:** 1") {
		t.Errorf("expected 1 passed test in summary")
	}
	if !strings.Contains(content, "* **Failed:** 1") {
		t.Errorf("expected 1 failed test in summary")
	}
	if !strings.Contains(content, "⚠️ Partially Passed") || !strings.Contains(content, "test 1") {
		t.Errorf("expected test 1 to be marked as partially passed")
	}
	if !strings.Contains(content, "1/2 passed.") {
		t.Errorf("expected 1/2 passed details")
	}
	if !strings.Contains(content, "| RUN | STATUS | DETAILS |") {
		t.Errorf("expected details table")
	}
	if !strings.Contains(content, "| 1 | ✅ PASSED | key1: value1 |") {
		t.Errorf("expected details for run 1")
	}
	if !strings.Contains(content, "| 2 | ❌ FAILED | error: some error |") {
		t.Errorf("expected details for run 2")
	}
	if !strings.Contains(content, "* **Total Duration:** 7.000 seconds") {
		t.Errorf("expected total duration 7.000s to be in summary")
	}
}
