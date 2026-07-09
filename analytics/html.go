package analytics

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/mtrace-project/mtrace/domain"
)

//go:embed templates/analytics.html
var htmlTemplateFS embed.FS

type htmlAnalyticsExporter struct {
	outputFolder string
	filename     string
	timestamp    time.Time
}

// htmlTemplateData is the data structure passed to the HTML template.
type htmlTemplateData struct {
	DataJSON template.JS
}

func (e *htmlAnalyticsExporter) Export(analytics []*TestAnalytics) error {
	jsonAnalyticsList := make([]jsonAnalytics, 0, len(analytics))
	for _, a := range analytics {
		if a == nil || a.TraceAnalytics == nil {
			continue
		}
		jsonAnalyticsList = append(jsonAnalyticsList, jsonAnalytics{
			TestName:  a.TestName,
			Timestamp: e.timestamp.Format(domain.TEXT_DATE_FORMAT),
			TraceAnalytics: jsonTraceAnalytics{
				MinDuration:               a.TraceAnalytics.MinDuration,
				MaxDuration:               a.TraceAnalytics.MaxDuration,
				P50Duration:               a.TraceAnalytics.P50Duration,
				P90Duration:               a.TraceAnalytics.P90Duration,
				P99Duration:               a.TraceAnalytics.P99Duration,
				DurationStandardDeviation: a.TraceAnalytics.DurationStandardDeviation,
				AverageDuration:           a.TraceAnalytics.AverageDuration,
				AverageSpanCount:          a.TraceAnalytics.AverageSpanCount,
				AverageSpanErrorCount:     a.TraceAnalytics.AverageSpanErrorCount,
				ErrorRate:                 a.TraceAnalytics.ErrorRate,
				SpanAnalytics:             spanAnalyticsToJSON(a.TraceAnalytics.SpanAnalytics),
			},
			Traces: tracesToJSON(a.Traces),
		})
	}

	dataBytes, err := json.Marshal(jsonAnalyticsList)
	if err != nil {
		return fmt.Errorf("failed to marshal analytics data: %w", err)
	}

	tmpl, err := template.ParseFS(htmlTemplateFS, "templates/analytics.html")
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	err = os.MkdirAll(e.outputFolder, PERM_DIR_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fullPath := filepath.Join(e.outputFolder, e.filename)

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, PERM_FILE_STANDARD)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	templateData := htmlTemplateData{
		DataJSON: template.JS(dataBytes),
	}

	err = tmpl.Execute(file, templateData)
	if err != nil {
		return fmt.Errorf("failed to render HTML template: %w", err)
	}

	slog.Info("HTML analytics exported successfully", "path", fullPath)

	return nil
}

func (e *htmlAnalyticsExporter) Format() string {
	return HTML_FORMAT
}

func newHTMLAnalyticsExporter(timestamp time.Time, outputFolder, filename string) *htmlAnalyticsExporter {
	return &htmlAnalyticsExporter{
		outputFolder: outputFolder,
		filename:     filename,
		timestamp:    timestamp,
	}
}
