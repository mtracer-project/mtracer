package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mtrace-project/mtrace/test"
)

func TestJSONExporter_Export(t *testing.T) {
	tempDir := t.TempDir()
	exporter := newJSONExporter(tempDir, "test.json", time.Now())

	results := []*test.TestSuite{
		test.NewTestSuite("test 1", []*test.TestResult{
			{
				Passed:   true,
				Duration: 5 * time.Second,
				Args:     []any{"key1", "value1", "key2", "value2"},
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

	expectedPath := filepath.Join(tempDir, "test.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file %s to be created", expectedPath)
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}

	var jsonResults jsonTestResults
	err = json.Unmarshal(data, &jsonResults)
	if err != nil {
		t.Fatalf("unexpected error unmarshaling file: %v", err)
	}

	if jsonResults.Duration != "7.000s" {
		t.Errorf("expected total duration 7.000s, got %s", jsonResults.Duration)
	}

	if len(jsonResults.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(jsonResults.Results))
	}

	if jsonResults.Results[0].TestName != "test 1" || !jsonResults.Results[0].Results[0].Passed {
		t.Errorf("expected test 1 to pass")
	}

	if jsonResults.Results[0].Results[0].Duration != "5.000s" {
		t.Errorf("expected test 1 duration 5.000s, got %s", jsonResults.Results[0].Results[0].Duration)
	}

	if jsonResults.Results[1].TestName != "test 2" || jsonResults.Results[1].Results[0].Passed {
		t.Errorf("expected test 2 to fail")
	}

	if jsonResults.Results[1].Results[0].Duration != "2.000s" {
		t.Errorf("expected test 2 duration 2.000s, got %s", jsonResults.Results[1].Results[0].Duration)
	}

	if jsonResults.Results[0].Results[0].Details["key1"] != "value1" {
		t.Errorf("expected key1 to be value1, got %v", jsonResults.Results[0].Results[0].Details["key1"])
	}
}
