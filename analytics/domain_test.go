package analytics

import (
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/span"
	"github.com/mtracer-project/mtracer/test"
	"github.com/mtracer-project/mtracer/trace"
)

func TestBuild(t *testing.T) {
	suites := []*test.TestSuite{
		nil,
		{
			Name: "test-1",
			Results: []*test.TestResult{
				{
					Trace: &trace.Trace{
						Duration:   100 * time.Millisecond,
						SpanCount:  2,
						ErrorCount: 0,
						Spans: []*span.Span{
							{
								ServiceName:   "srv1",
								OperationName: "op1",
								Duration:      50 * time.Millisecond,
								SpanStatus:    "ok",
							},
							{
								ServiceName:   "srv1",
								OperationName: "op2",
								Duration:      50 * time.Millisecond,
								SpanStatus:    "ok",
							},
						},
					},
				},
				{
					Trace: &trace.Trace{
						Duration:   200 * time.Millisecond,
						SpanCount:  2,
						ErrorCount: 1,
						Spans: []*span.Span{
							{
								ServiceName:   "srv1",
								OperationName: "op1",
								Duration:      150 * time.Millisecond,
								SpanStatus:    "error",
							},
							{
								ServiceName:   "srv1",
								OperationName: "op2",
								Duration:      50 * time.Millisecond,
								SpanStatus:    "ok",
							},
						},
					},
				},
			},
		},
		{
			Name: "test-2",
			Results: []*test.TestResult{
				{
					Trace: nil,
				},
			},
		},
	}

	analyticsList := Build(suites)

	if len(analyticsList) != 2 {
		t.Fatalf("expected 2 test analytics, got %d", len(analyticsList))
	}

	// Verify test-1 analytics
	a1 := analyticsList[0]
	if a1.TestName != "test-1" {
		t.Errorf("expected test name 'test-1', got %s", a1.TestName)
	}
	if a1.TraceAnalytics == nil {
		t.Fatalf("expected TraceAnalytics to be populated")
	}

	ta := a1.TraceAnalytics
	if ta.MinDuration != 100 {
		t.Errorf("expected MinDuration 100, got %d", ta.MinDuration)
	}
	if ta.MaxDuration != 200 {
		t.Errorf("expected MaxDuration 200, got %d", ta.MaxDuration)
	}
	if ta.AverageDuration != 150.0 {
		t.Errorf("expected AverageDuration 150.0, got %f", ta.AverageDuration)
	}
	if ta.AverageSpanCount != 2.0 {
		t.Errorf("expected AverageSpanCount 2.0, got %f", ta.AverageSpanCount)
	}
	if ta.ErrorRate != 50.0 {
		t.Errorf("expected ErrorRate 50.0, got %f", ta.ErrorRate)
	}

	// Verify spans logic within test-1
	if len(ta.SpanAnalytics) != 2 {
		t.Fatalf("expected 2 SpanAnalytics keys, got %d", len(ta.SpanAnalytics))
	}
	s1 := ta.SpanAnalytics["srv1-op1"]
	if s1.ServiceName != "srv1" || s1.OperationName != "op1" {
		t.Errorf("expected srv1-op1, got %s-%s", s1.ServiceName, s1.OperationName)
	}
	if s1.MinDuration != 50 || s1.MaxDuration != 150 {
		t.Errorf("expected Min/Max 50/150, got %d/%d", s1.MinDuration, s1.MaxDuration)
	}
	if s1.ErrorRate != 50.0 {
		t.Errorf("expected ErrorRate 50.0, got %f", s1.ErrorRate)
	}

	// Verify test-2 analytics (nil trace)
	a2 := analyticsList[1]
	if a2.TestName != "test-2" {
		t.Errorf("expected test name 'test-2', got %s", a2.TestName)
	}
	// It should gracefully calculate with empty slices because we added fixes
	if a2.TraceAnalytics == nil {
		t.Fatalf("expected TraceAnalytics to not be nil even for empty data")
	}
}
