package export

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/test"
)

type jsonExporter struct {
	outputFolder string
	filename     string
	timestamp    time.Time
}

type jsonTestResults struct {
	Timestamp string         `json:"timestamp"`
	Duration  string         `json:"duration"`
	Results   []jsonTestInfo `json:"results"`
}

type jsonTestInfo struct {
	TestName string           `json:"testName"`
	Results  []jsonTestResult `json:"results"`
}

type jsonTestResult struct {
	Run      int            `json:"run"`
	Passed   bool           `json:"passed"`
	Duration string         `json:"duration"`
	Details  map[string]any `json:"details"`
}

func (e *jsonExporter) Export(suites []*test.TestSuite) error {
	jsonResults := jsonTestResults{
		Timestamp: e.timestamp.Format(domain.DATE_FORMAT),
	}
	var totalTime time.Duration
	for _, suite := range suites {
		testResults := make([]jsonTestResult, 0, len(suite.Results))
		for i, r := range suite.Results {
			totalTime += r.Duration
			details := make(map[string]any)

			for j := 0; j < len(r.Args); j += 2 {
				if j+1 < len(r.Args) {
					key := fmt.Sprintf("%v", r.Args[j])
					value := r.Args[j+1]
					details[key] = value
				}
			}

			jsonResult := jsonTestResult{
				Run:      i + 1,
				Passed:   r.Passed,
				Duration: fmt.Sprintf("%.3fs", r.Duration.Seconds()),
				Details:  details,
			}
			testResults = append(testResults, jsonResult)
		}

		jsonInfo := jsonTestInfo{
			TestName: suite.Name,
			Results:  testResults,
		}
		jsonResults.Results = append(jsonResults.Results, jsonInfo)
	}

	jsonResults.Duration = fmt.Sprintf("%.3fs", totalTime.Seconds())

	jsonResultsBytes, err := json.MarshalIndent(jsonResults, "", "    ")
	if err != nil {
		return err
	}

	// Create the output directory if it doesn't exist
	err = os.MkdirAll(e.outputFolder, PERM_DIR_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	fullPath := filepath.Join(e.outputFolder, e.filename)

	// Write the JSON results to the specified file
	err = os.WriteFile(fullPath, jsonResultsBytes, PERM_FILE_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to write json file: %w", err)
	}

	slog.Info("JSON report exported successfully", "path", fullPath)

	return nil
}

func (e *jsonExporter) Format() string {
	return JSON_FORMAT
}

func newJSONExporter(outputFolder, filename string, timestamp time.Time) *jsonExporter {
	return &jsonExporter{
		outputFolder: outputFolder,
		filename:     filename,
		timestamp:    timestamp,
	}
}
