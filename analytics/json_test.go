package analytics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/span"
	"github.com/mtrace-project/mtrace/trace"
	"github.com/mtrace-project/mtrace/trigger"
)

func TestJSONAnalyticsExporter_Format(t *testing.T) {
	exporter := newJSONAnalyticsExporter(time.Now(), "output", "test.json")
	if exporter.Format() != JSON_FORMAT {
		t.Errorf("expected format %q, got %q", JSON_FORMAT, exporter.Format())
	}
}

func TestJSONAnalyticsExporter_Export_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "json-analytics-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	timestamp := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	filename := "analytics.json"
	exporter := newJSONAnalyticsExporter(timestamp, tempDir, filename)

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
			TestName: "test-1",
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
		nil, // should be skipped
		{
			TestName:       "test-nil-traceanalytics",
			TraceAnalytics: nil, // should be skipped
		},
	}

	err = exporter.Export(analyticsData)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	fullPath := filepath.Join(tempDir, filename)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read generated json file: %v", err)
	}

	var results []jsonAnalytics
	err = json.Unmarshal(data, &results)
	if err != nil {
		t.Fatalf("failed to unmarshal generated json: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.TestName != "test-1" {
		t.Errorf("expected testName test-1, got %s", res.TestName)
	}
	if res.Timestamp != timestamp.Format(domain.DATE_FORMAT) {
		t.Errorf("expected timestamp %s, got %s", timestamp.Format(domain.DATE_FORMAT), res.Timestamp)
	}

	// Validate TraceAnalytics
	if res.TraceAnalytics.AverageDuration != 200.0 {
		t.Errorf("expected averageDuration 200.0, got %f", res.TraceAnalytics.AverageDuration)
	}
	if res.TraceAnalytics.P99Duration != 290 {
		t.Errorf("expected p99Duration 290, got %d", res.TraceAnalytics.P99Duration)
	}

	// Validate SpanAnalytics
	if len(res.TraceAnalytics.SpanAnalytics) != 1 {
		t.Fatalf("expected 1 span analytics, got %d", len(res.TraceAnalytics.SpanAnalytics))
	}
	sa := res.TraceAnalytics.SpanAnalytics[0]
	if sa.ServiceName != "serviceA" || sa.OperationName != "op1" {
		t.Errorf("expected span serviceA-op1, got %s-%s", sa.ServiceName, sa.OperationName)
	}
	if sa.AveragePercentageOfTotalTrace != 50.0 {
		t.Errorf("expected averagePercentageOfTotalTrace 50.0, got %f", sa.AveragePercentageOfTotalTrace)
	}

	// Validate Traces
	if len(res.Traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(res.Traces))
	}
	tr := res.Traces[0]
	if tr.TraceId != "12345678901234567890123456789012" {
		t.Errorf("expected traceId 12345678901234567890123456789012, got %s", tr.TraceId)
	}
	if len(tr.Spans) != 1 {
		t.Fatalf("expected 1 span in trace, got %d", len(tr.Spans))
	}
	sp := tr.Spans[0]
	if sp.SpanId != "span1" {
		t.Errorf("expected spanId span1, got %s", sp.SpanId)
	}
}

func TestJSONAnalyticsExporter_Export_DirCreationError(t *testing.T) {
	// Create an exporter pointing to an invalid directory path to trigger os.MkdirAll error
	exporter := newJSONAnalyticsExporter(time.Now(), "/invalid_dir\x00_that_fails/test", "test.json")

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

func TestSpanAnalyticsToJSON_Sorting(t *testing.T) {
	input := map[string]*SpanAnalytics{
		"ZService-ZOp": {
			ServiceName:     "ZService",
			OperationName:   "ZOp",
			AveragePosition: 3.5,
		},
		"AService-BOp": {
			ServiceName:     "AService",
			OperationName:   "BOp",
			AveragePosition: 1.2,
		},
		"AService-AOp": {
			ServiceName:     "AService",
			OperationName:   "AOp",
			AveragePosition: 2.1,
		},
		"BService-BOp": {
			ServiceName:     "BService",
			OperationName:   "BOp",
			AveragePosition: 1.2, // Same as AService-BOp to test secondary alphabetical sort
		},
	}

	result := spanAnalyticsToJSON(input)

	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	if result[0].ServiceName != "AService" || result[0].OperationName != "BOp" {
		t.Errorf("expected first element to be AService-BOp (pos 1.2), got %s-%s", result[0].ServiceName, result[0].OperationName)
	}
	if result[1].ServiceName != "BService" || result[1].OperationName != "BOp" {
		t.Errorf("expected second element to be BService-BOp (pos 1.2), got %s-%s", result[1].ServiceName, result[1].OperationName)
	}
	if result[2].ServiceName != "AService" || result[2].OperationName != "AOp" {
		t.Errorf("expected third element to be AService-AOp (pos 2.1), got %s-%s", result[2].ServiceName, result[2].OperationName)
	}
	if result[3].ServiceName != "ZService" || result[3].OperationName != "ZOp" {
		t.Errorf("expected fourth element to be ZService-ZOp (pos 3.5), got %s-%s", result[3].ServiceName, result[3].OperationName)
	}
}

func TestTracesToJSON_NilHandling(t *testing.T) {
	traces := []*trace.Trace{
		nil,
		{
			TraceId: trigger.TraceId("12345678901234567890123456789012"),
			Spans: []*span.Span{
				nil,
				{
					SpanId: "span1",
				},
			},
		},
	}

	result := tracesToJSON(traces)
	if len(result) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(result))
	}

	if len(result[0].Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(result[0].Spans))
	}
}
