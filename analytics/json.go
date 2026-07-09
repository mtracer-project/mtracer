package analytics

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/trace"
)

type jsonAnalyticsExporter struct {
	outputFolder string
	filename     string
	timestamp    time.Time
}

func (e *jsonAnalyticsExporter) Export(analytics []*TestAnalytics) error {
	jsonAnalyticsList := make([]jsonAnalytics, 0, len(analytics))
	for _, a := range analytics {
		if a == nil || a.TraceAnalytics == nil {
			continue
		}
		jsonAnalyticsList = append(jsonAnalyticsList, jsonAnalytics{
			TestName:  a.TestName,
			Timestamp: e.timestamp.Format(domain.DATE_FORMAT),
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

	jsonResultsBytes, err := json.MarshalIndent(jsonAnalyticsList, "", "    ")
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

	slog.Info("JSON analytics exported successfully", "path", fullPath)

	return nil
}

func spanAnalyticsToJSON(spanAnalytics map[string]*SpanAnalytics) []jsonSpanAnalytics {
	jsonSpanAnalyticsList := make([]jsonSpanAnalytics, 0, len(spanAnalytics))
	for _, sa := range spanAnalytics {
		jsonSpanAnalyticsList = append(jsonSpanAnalyticsList, jsonSpanAnalytics{
			ServiceName:                   sa.ServiceName,
			OperationName:                 sa.OperationName,
			MaxDuration:                   sa.MaxDuration,
			MinDuration:                   sa.MinDuration,
			P50Duration:                   sa.P50Duration,
			P90Duration:                   sa.P90Duration,
			P99Duration:                   sa.P99Duration,
			DurationStandardDeviation:     sa.DurationStandardDeviation,
			AverageDuration:               sa.AverageDuration,
			ErrorRate:                     sa.ErrorRate,
			AveragePercentageOfTotalTrace: sa.AveragePercentageOfTotalTrace,
			AveragePosition:               sa.AveragePosition,
		})
	}

	// Sort the slice based on AveragePosition
	slices.SortFunc(jsonSpanAnalyticsList, func(a, b jsonSpanAnalytics) int {
		if a.AveragePosition == b.AveragePosition {
			if a.ServiceName == b.ServiceName {
				return strings.Compare(a.OperationName, b.OperationName)
			}
			return strings.Compare(a.ServiceName, b.ServiceName)
		}
		if a.AveragePosition < b.AveragePosition {
			return -1
		}
		return 1
	})

	return jsonSpanAnalyticsList
}

func tracesToJSON(traces []*trace.Trace) []jsonTrace {
	jsonTraces := make([]jsonTrace, 0, len(traces))

	for _, t := range traces {
		if t == nil {
			continue
		}
		jsonSpans := make([]jsonSpan, 0, len(t.Spans))
		for _, s := range t.Spans {
			if s == nil {
				continue
			}
			jsonSpans = append(jsonSpans, jsonSpan{
				SpanId:        s.SpanId,
				ParentId:      s.ParentId,
				ServiceName:   s.ServiceName,
				OperationName: s.OperationName,
				SpanKind:      s.SpanKind,
				SpanStatus:    s.SpanStatus,
				StartTime:     s.StartTime.Format(domain.DATE_FORMAT),
				EndTime:       s.EndTime.Format(domain.DATE_FORMAT),
				Duration:      s.Duration.Milliseconds(),
			})
		}

		jsonTraces = append(jsonTraces, jsonTrace{
			TraceId:    t.TraceId.String(),
			StartTime:  t.StartTime.Format(domain.DATE_FORMAT),
			EndTime:    t.EndTime.Format(domain.DATE_FORMAT),
			Duration:   t.Duration.Milliseconds(),
			SpanCount:  len(t.Spans),
			ErrorCount: t.ErrorCount,
			Spans:      jsonSpans,
		})
	}

	return jsonTraces
}

func (e *jsonAnalyticsExporter) Format() string {
	return JSON_FORMAT
}

func newJSONAnalyticsExporter(timestamp time.Time, outputFolder, filename string) *jsonAnalyticsExporter {
	return &jsonAnalyticsExporter{
		outputFolder: outputFolder,
		filename:     filename,
		timestamp:    timestamp,
	}
}

type jsonAnalytics struct {
	TestName       string             `json:"testName"`
	Timestamp      string             `json:"timestamp"`
	TraceAnalytics jsonTraceAnalytics `json:"traceAnalytics"`
	Traces         []jsonTrace        `json:"traces"`
}

type jsonTraceAnalytics struct {
	MinDuration int64 `json:"minDuration"` // Minimum trace duration in milliseconds
	MaxDuration int64 `json:"maxDuration"`
	P50Duration int64 `json:"p50Duration"` // 50th percentile trace duration in milliseconds
	P90Duration int64 `json:"p90Duration"`
	P99Duration int64 `json:"p99Duration"`

	DurationStandardDeviation float64 `json:"durationStandardDeviation"`
	AverageDuration           float64 `json:"averageDuration"`

	AverageSpanCount      float64 `json:"averageSpanCount"`
	AverageSpanErrorCount float64 `json:"averageSpanErrorCount"`
	ErrorRate             float64 `json:"errorRate"` // Percentage of traces with at least one span with an error status

	SpanAnalytics []jsonSpanAnalytics `json:"spanAnalytics"`
}

type jsonSpanAnalytics struct {
	ServiceName                   string  `json:"serviceName"`
	OperationName                 string  `json:"operationName"`
	MaxDuration                   int64   `json:"maxDuration"`
	MinDuration                   int64   `json:"minDuration"`
	P50Duration                   int64   `json:"p50Duration"`
	P90Duration                   int64   `json:"p90Duration"`
	P99Duration                   int64   `json:"p99Duration"`
	DurationStandardDeviation     float64 `json:"durationStandardDeviation"`
	AverageDuration               float64 `json:"averageDuration"`
	ErrorRate                     float64 `json:"errorRate"`
	AveragePercentageOfTotalTrace float64 `json:"averagePercentageOfTotalTrace"` // Average percentage of the span duration over the total trace duration
	AveragePosition               float64 `json:"-"`
}

type jsonTrace struct {
	TraceId    string     `json:"traceId"`
	StartTime  string     `json:"startTime"`
	EndTime    string     `json:"endTime"`
	Duration   int64      `json:"duration"` // Duration in milliseconds
	SpanCount  int        `json:"spanCount"`
	ErrorCount int        `json:"errorCount"`
	Spans      []jsonSpan `json:"spans"`
}

type jsonSpan struct {
	SpanId        string `json:"spanId"`
	ParentId      string `json:"parentId"`
	ServiceName   string `json:"serviceName"`
	OperationName string `json:"operationName"`
	SpanKind      string `json:"spanKind"`
	SpanStatus    string `json:"spanStatus"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	Duration      int64  `json:"duration"` // Duration in milliseconds

	// Attributes map[string]any `json:"attributes"` // the json file would be too verbose if we include all span attributes
}
