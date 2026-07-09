package export

import (
	"fmt"
	"strings"

	"github.com/mtrace-project/mtrace/test"
)

func formatDetails(result *test.TestResult) string {
	if result == nil {
		return ""
	}
	var details []string

	for i := 0; i < len(result.Args); i += 2 {
		if i+1 < len(result.Args) {
			key := fmt.Sprintf("%v", result.Args[i])
			value := fmt.Sprintf("%v", result.Args[i+1])
			details = append(details, fmt.Sprintf("%s: %s", key, value))
		}
	}

	return strings.Join(details, " | ")
}
