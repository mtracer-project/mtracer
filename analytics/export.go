package analytics

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mtracer-project/mtracer/domain"
)

const (
	JSON_FORMAT = "json"
	HTML_FORMAT = "html"

	// PERM_DIR_STANDARD stands for (rwxr-xr-x): read and execute for everyone, write only for the owner
	PERM_DIR_STANDARD os.FileMode = 0o755
	// PERM_FILE_STANDARD stands for (rw-r--r--): read for everyone, write only for the owner
	PERM_FILE_STANDARD os.FileMode = 0o644
)

type AnalyticsExporter interface {
	Export(analytics []*TestAnalytics) error
	Format() string
}

func NewAnalyticsExporters(formats []string, baseDir string) ([]AnalyticsExporter, error) {
	exporters := make(map[string]AnalyticsExporter, len(formats))
	timestamp := time.Now()
	for _, format := range formats {
		folderPath := filepath.Join(baseDir, fmt.Sprintf("%s_analytics_exports", format))
		switch format {
		case JSON_FORMAT:
			exporters[format] = newJSONAnalyticsExporter(timestamp, folderPath, fmt.Sprintf("%s_analytics.json", timestamp.Format(domain.FILENAME_DATE_FORMAT)))
		case HTML_FORMAT:
			exporters[format] = newHTMLAnalyticsExporter(timestamp, folderPath, fmt.Sprintf("%s_analytics.html", timestamp.Format(domain.FILENAME_DATE_FORMAT)))
		default:
			return nil, fmt.Errorf("unsupported analytics format: %s", format)
		}
	}

	exportersList := make([]AnalyticsExporter, 0, len(exporters))
	for _, exporter := range exporters {
		exportersList = append(exportersList, exporter)
	}

	return exportersList, nil
}
