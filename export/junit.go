package export

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jstemmer/go-junit-report/v2/junit"
	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/test"
)

type junitExporter struct {
	outputFolder string
	filename     string
	timestamp    time.Time
}

func (e *junitExporter) Export(suites []*test.TestSuite) error {
	juSuites := make([]junit.Testsuite, 0, len(suites))

	for i, suite := range suites {
		testcases := make([]junit.Testcase, 0, len(suite.Results))
		var failures int
		var totalTime time.Duration

		for i, r := range suite.Results {
			totalTime += r.Duration

			tc := junit.Testcase{
				Name:      fmt.Sprintf("%s (Run %d)", suite.Name, i+1),
				Classname: strings.ReplaceAll(suite.Name, " ", "-"),
				Time:      fmt.Sprintf("%.3f", r.Duration.Seconds()),
			}
			if !r.Passed {
				failures++
				tc.Failure = &junit.Result{
					Message: "Test failed",
					Data:    formatDetails(r),
				}
			}
			testcases = append(testcases, tc)
		}

		juSuites = append(juSuites, junit.Testsuite{
			ID:        i + 1,
			Name:      suite.Name,
			Tests:     len(testcases),
			Errors:    0,
			Failures:  failures,
			Testcases: testcases,
			Timestamp: e.timestamp.Format(domain.DATE_FORMAT),
			Time:      fmt.Sprintf("%.3f", totalTime.Seconds()),
		})
	}

	testsuites := &junit.Testsuites{
		Suites: juSuites,
	}

	xmlBytes, err := xml.MarshalIndent(testsuites, "", "    ")
	if err != nil {
		return err
	}

	// Create the output directory if it doesn't exist
	err = os.MkdirAll(e.outputFolder, PERM_DIR_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	fullPath := filepath.Join(e.outputFolder, e.filename)

	err = os.WriteFile(fullPath, append([]byte(xml.Header), xmlBytes...), PERM_FILE_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to write junit file: %w", err)
	}

	slog.Info("JUnit report exported successfully", "path", fullPath)

	return nil
}

func (e *junitExporter) Format() string {
	return JUNIT_FORMAT
}

func newJunitExporter(outputFolder, filename string, timestamp time.Time) *junitExporter {
	return &junitExporter{
		outputFolder: outputFolder,
		filename:     filename,
		timestamp:    timestamp,
	}
}
