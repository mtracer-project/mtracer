package export

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/test"
)

type markdownExporter struct {
	outputFolder string
	filename     string
	runCount     int
	timestamp    time.Time
}

func (e *markdownExporter) Export(suites []*test.TestSuite) error {
	content := ""
	if e.runCount > 1 {
		content = e.multipleRunsMarkdownSummary(suites)
	} else {
		content = e.markdownSummary(suites)
	}

	// Create the output directory if it doesn't exist
	err := os.MkdirAll(e.outputFolder, PERM_DIR_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	fullPath := filepath.Join(e.outputFolder, e.filename)

	err = os.WriteFile(fullPath, []byte(content), PERM_FILE_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	slog.Info("Markdown report exported successfully", "path", fullPath)

	return nil
}

func (e *markdownExporter) multipleRunsMarkdownSummary(suites []*test.TestSuite) string {
	var sb strings.Builder

	passedCount := 0
	failedCount := 0
	var totalTime time.Duration
	for _, suite := range suites {
		for _, r := range suite.Results {
			totalTime += r.Duration
			if r.Passed {
				passedCount++
			} else {
				failedCount++
			}
		}
	}

	sb.WriteString("# Test Results Report\n\n")
	fmt.Fprintf(&sb, "**Timestamp:** %s\n\n", e.timestamp.Format(domain.DATE_FORMAT))
	sb.WriteString("## Summary\n")
	fmt.Fprintf(&sb, "* **Total Tests:** %d\n", len(suites))
	fmt.Fprintf(&sb, "* **Total Runs:** %d\n", passedCount+failedCount)
	fmt.Fprintf(&sb, "* **Passed:** %d\n", passedCount)
	fmt.Fprintf(&sb, "* **Failed:** %d\n\n", failedCount)
	fmt.Fprintf(&sb, "* **Total Duration:** %.3f seconds\n\n", totalTime.Seconds())

	sb.WriteString("## Detailed Results\n\n")

	for _, suite := range suites {
		if len(suite.Results) == 0 {
			continue
		}

		testPassed := 0
		var testTime time.Duration
		for _, r := range suite.Results {
			testTime += r.Duration
			if r.Passed {
				testPassed++
			}
		}

		statusIcon := "❌ Failed"
		if testPassed == len(suite.Results) {
			statusIcon = "✅ Passed"
		} else if testPassed > 0 {
			statusIcon = "⚠️ Partially Passed"
		}

		fmt.Fprintf(&sb, "### %s %s\n", statusIcon, suite.Name)
		fmt.Fprintf(&sb, "%d/%d passed.\n\n", testPassed, len(suite.Results))
		fmt.Fprintf(&sb, "* **Total Duration:** %.3f seconds\n\n", testTime.Seconds())

		sb.WriteString("| RUN | STATUS | DETAILS |\n")
		sb.WriteString("| --- | ------ | ------- |\n")

		for i, r := range suite.Results {
			if r == nil {
				continue
			}
			status := "❌ FAILED"
			if r.Passed {
				status = "✅ PASSED"
			}
			details := formatDetails(r)
			if details == "" {
				details = "-"
			}
			fmt.Fprintf(&sb, "| %d | %s | %s |\n", i+1, status, details)
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func (e *markdownExporter) markdownSummary(suites []*test.TestSuite) string {
	var sb strings.Builder

	passedCount := 0
	failedCount := 0
	var totalTime time.Duration
	for _, r := range suites {
		if len(r.Results) == 0 {
			continue
		}
		totalTime += r.Results[0].Duration
		if r.Results[0].Passed {
			passedCount++
		} else {
			failedCount++
		}
	}

	sb.WriteString("# Test Results Report\n\n")
	fmt.Fprintf(&sb, "**Timestamp:** %s\n\n", e.timestamp.Format(domain.TEXT_DATE_FORMAT))
	sb.WriteString("## Summary\n")
	fmt.Fprintf(&sb, "* **Total Tests:** %d\n", len(suites))
	fmt.Fprintf(&sb, "* **Passed:** %d\n", passedCount)
	fmt.Fprintf(&sb, "* **Failed:** %d\n\n", failedCount)
	fmt.Fprintf(&sb, "* **Total Duration:** %.3f seconds\n\n", totalTime.Seconds())

	sb.WriteString("## Detailed Results\n\n")

	for _, r := range suites {
		if len(r.Results) == 0 {
			continue
		}

		statusIcon := "❌ Failed"
		if r.Results[0].Passed {
			statusIcon = "✅ Passed"
			fmt.Fprintf(&sb, "### %s %s\n", statusIcon, r.Name)
			sb.WriteString("Test executed successfully.\n")
		} else {
			fmt.Fprintf(&sb, "### %s %s\n", statusIcon, r.Name)
			sb.WriteString("Test failed.\n")
		}

		fmt.Fprintf(&sb, "* **Duration:** %.3f seconds\n", r.Results[0].Duration.Seconds())

		details := formatDetails(r.Results[0])
		if details != "" {
			sb.WriteString("\n**Details:**.\n")
			for _, d := range strings.Split(details, " | ") {
				fmt.Fprintf(&sb, "- %s\n", d)
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func (e *markdownExporter) Format() string {
	return MARKDOWN_FORMAT
}

func newMarkdownExporter(runCount int, outputFolder, filename string, timestamp time.Time) *markdownExporter {
	return &markdownExporter{
		runCount:     runCount,
		outputFolder: outputFolder,
		filename:     filename,
		timestamp:    timestamp,
	}
}
