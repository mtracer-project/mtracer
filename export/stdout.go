package export

import (
	"fmt"
	"os"

	"github.com/mtrace-project/mtrace/test"
	"github.com/olekukonko/tablewriter"
)

type stdoutExporter struct {
	runCount int
}

func (e *stdoutExporter) Export(suites []*test.TestSuite) error {
	if e.runCount > 1 {
		return displayMultipleRunsSummary(suites, "PASSED", "FAILED")
	}
	return DisplayTestsSummary(suites, "PASSED", "FAILED")
}

func (e *stdoutExporter) Format() string {
	return STDOUT_FORMAT
}

const (
	COLOR_RESET = "\033[0m"
	COLOR_GREEN = "\033[32m"
	COLOR_RED   = "\033[31m"
)

func displayMultipleRunsSummary(suites []*test.TestSuite, okStatus, notOkStatus string) error {
	if len(suites) == 0 {
		return nil
	}

	for _, suite := range suites {
		fmt.Printf("Test: %s\n", suite.Name) // Print the test name before the table
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"RUN", "STATUS", "DETAILS"})
		table.SetAutoFormatHeaders(false)

		if len(suite.Results) == 0 {
			table.Append([]string{"0", notOkStatus, "No results available"})
			continue
		}

		for i, result := range suite.Results {
			status := notOkStatus
			colorCode := COLOR_RED
			if result.Passed {
				status = okStatus
				colorCode = COLOR_GREEN
			}
			statusColored := colorCode + status + COLOR_RESET

			details := formatDetails(result)

			table.Append([]string{fmt.Sprintf("%d", i+1), statusColored, details})
		}
		table.Render()
		fmt.Println() // Add a newline between test summaries
	}

	return nil
}

func DisplayTestsSummary(suites []*test.TestSuite, okStatus, notOkStatus string) error {
	if len(suites) == 0 {
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"TEST NAME", "STATUS", "DETAILS"})
	table.SetAutoFormatHeaders(false)

	for _, suite := range suites {
		if len(suite.Results) == 0 {
			table.Append([]string{suite.Name, notOkStatus, "No results available"})
			continue
		}
		status := notOkStatus
		colorCode := COLOR_RED
		if suite.Results[0].Passed {
			status = okStatus
			colorCode = COLOR_GREEN
		}
		statusColored := colorCode + status + COLOR_RESET

		details := formatDetails(suite.Results[0])

		table.Append([]string{suite.Name, statusColored, details})
	}

	table.Render()
	return nil
}

func newStdoutExporter(runCount int) *stdoutExporter {
	return &stdoutExporter{
		runCount: runCount,
	}
}
