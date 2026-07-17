package export

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/test"
)

const (
	JSON_FORMAT     = "json"
	JUNIT_FORMAT    = "junit"
	MARKDOWN_FORMAT = "markdown"
	STDOUT_FORMAT   = "stdout"

	// PERM_DIR_STANDARD stands for (rwxr-xr-x): read and execute for everyone, write only for the owner
	PERM_DIR_STANDARD os.FileMode = 0o755
	// PERM_FILE_STANDARD stands for (rw-r--r--): read for everyone, write only for the owner
	PERM_FILE_STANDARD os.FileMode = 0o644
)

type Exporter interface {
	Export(suites []*test.TestSuite) error
	Format() string
}

func NewExporters(formats []string, runCount int, baseDir string) ([]Exporter, error) {
	exporters := make(map[string]Exporter, len(formats))
	timestamp := time.Now()
	for _, format := range formats {
		if format == "md" {
			format = MARKDOWN_FORMAT
		}
		folderPath := filepath.Join(baseDir, fmt.Sprintf("%s_exports", format))
		switch format {
		case JSON_FORMAT:
			exporters[format] = newJSONExporter(folderPath, fmt.Sprintf("%s_test_results.json", timestamp.Format(domain.FILENAME_DATE_FORMAT)), timestamp)
		case JUNIT_FORMAT:
			exporters[format] = newJunitExporter(folderPath, fmt.Sprintf("%s_test_results.xml", timestamp.Format(domain.FILENAME_DATE_FORMAT)), timestamp)
		case MARKDOWN_FORMAT:
			exporters[format] = newMarkdownExporter(runCount, folderPath, fmt.Sprintf("%s_test_results.md", timestamp.Format(domain.FILENAME_DATE_FORMAT)), timestamp)
		case STDOUT_FORMAT:
			exporters[format] = newStdoutExporter(runCount)
		default:
			return nil, fmt.Errorf("unsupported export format: %s", format)
		}
	}

	exportersList := make([]Exporter, 0, len(exporters))
	for _, exporter := range exporters {
		exportersList = append(exportersList, exporter)
	}

	return exportersList, nil
}
