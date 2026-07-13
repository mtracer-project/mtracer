package analytics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/span"
	"github.com/mtracer-project/mtracer/trace"
	"github.com/mtracer-project/mtracer/trigger"
)

func TestHTMLAnalyticsExporter_Format(t *testing.T) {
	exporter := newHTMLAnalyticsExporter(time.Now(), "output", "test.html")
	if exporter.Format() != HTML_FORMAT {
		t.Errorf("expected format %q, got %q", HTML_FORMAT, exporter.Format())
	}
}

func TestHTMLAnalyticsExporter_Export_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "html-analytics-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	timestamp := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	filename := "analytics.html"
	exporter := newHTMLAnalyticsExporter(timestamp, tempDir, filename)

	testTraceId, _ := trigger.NewTraceId("12345678901234567890123456789012")

	span1 := &span.Span{
		SpanId:        "span1",
		ServiceName:   "serviceA",
		OperationName: "op1",
		SpanKind:      "server",
		SpanStatus:    "ok",
		StartTime:     timestamp,
		EndTime:       timestamp.Add(100 * time.Millisecond),
		Duration:      100 * time.Millisecond,
	}

	trace1 := &trace.Trace{
		TraceId:    testTraceId,
		StartTime:  timestamp,
		EndTime:    timestamp.Add(200 * time.Millisecond),
		Duration:   200 * time.Millisecond,
		SpanCount:  1,
		ErrorCount: 0,
		Spans:      []*span.Span{span1},
	}

	analyticsData := []*TestAnalytics{
		{
			TestName: "test-suite-1",
			TraceAnalytics: &TraceAnalytics{
				MinDuration:               100,
				MaxDuration:               300,
				P50Duration:               200,
				P90Duration:               250,
				P99Duration:               290,
				DurationStandardDeviation: 15.5,
				AverageDuration:           200.0,
				AverageSpanCount:          2.5,
				AverageSpanErrorCount:     0.5,
				ErrorRate:                 10.0,
				SpanAnalytics: map[string]*SpanAnalytics{
					"serviceA-op1": {
						ServiceName:                   "serviceA",
						OperationName:                 "op1",
						MaxDuration:                   150,
						MinDuration:                   50,
						P50Duration:                   100,
						P90Duration:                   140,
						P99Duration:                   148,
						DurationStandardDeviation:     5.5,
						AverageDuration:               100.0,
						ErrorRate:                     5.0,
						AveragePercentageOfTotalTrace: 50.0,
					},
				},
			},
			Traces: []*trace.Trace{trace1},
		},
	}

	err = exporter.Export(analyticsData)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	fullPath := filepath.Join(tempDir, filename)
	data, err := os.ReadFile(fullPath) // nolint:gosec
	if err != nil {
		t.Fatalf("failed to read generated html file: %v", err)
	}

	content := string(data)

	// Verify the HTML has the expected structure
	expectations := []string{
		"<!DOCTYPE html>",
		"test-suite-1",
		"suite-select",
		"chart-duration",
		"chart-errors",
	}

	for _, expected := range expectations {
		if !strings.Contains(content, expected) {
			t.Errorf("expected HTML to contain %q", expected)
		}
	}
}

func TestHTMLAnalyticsExporter_Export_NilHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "html-analytics-nil-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	exporter := newHTMLAnalyticsExporter(time.Now(), tempDir, "test.html")

	analyticsData := []*TestAnalytics{
		nil,
		{
			TestName:       "test-nil-traceanalytics",
			TraceAnalytics: nil,
		},
	}

	err = exporter.Export(analyticsData)
	if err != nil {
		t.Fatalf("expected no error for nil analytics, got: %v", err)
	}

	// Verify the file was created (even with empty data)
	fullPath := filepath.Join(tempDir, "test.html")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatal("expected HTML file to be created")
	}
}

func TestHTMLAnalyticsExporter_Export_DirCreationError(t *testing.T) {
	exporter := newHTMLAnalyticsExporter(time.Now(), "/invalid_dir\x00_that_fails/test", "test.html")

	analyticsData := []*TestAnalytics{
		{
			TestName:       "test",
			TraceAnalytics: &TraceAnalytics{},
			Traces:         []*trace.Trace{},
		},
	}

	err := exporter.Export(analyticsData)
	if err == nil {
		t.Fatal("expected error when creating invalid directory, got nil")
	}
}

func TestHTMLAnalyticsExporter_Export_MultiSuite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "html-analytics-multi-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	exporter := newHTMLAnalyticsExporter(time.Now(), tempDir, "multi.html")

	analyticsData := []*TestAnalytics{
		{
			TestName: "suite-alpha",
			TraceAnalytics: &TraceAnalytics{
				MinDuration:   50,
				MaxDuration:   500,
				SpanAnalytics: map[string]*SpanAnalytics{},
			},
			Traces: []*trace.Trace{},
		},
		{
			TestName: "suite-beta",
			TraceAnalytics: &TraceAnalytics{
				MinDuration:   10,
				MaxDuration:   100,
				SpanAnalytics: map[string]*SpanAnalytics{},
			},
			Traces: []*trace.Trace{},
		},
	}

	err = exporter.Export(analyticsData)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tempDir, "multi.html")) // nolint:gosec
	if err != nil {
		t.Fatalf("failed to read generated html: %v", err)
	}

	content := string(data)

	// Both suite names should appear in the data
	if !strings.Contains(content, "suite-alpha") {
		t.Error("expected HTML to contain suite-alpha")
	}
	if !strings.Contains(content, "suite-beta") {
		t.Error("expected HTML to contain suite-beta")
	}
}
