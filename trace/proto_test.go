package trace_test

import (
	"testing"
	"time"

	"github.com/mtrace-project/mtrace/span"
	"github.com/mtrace-project/mtrace/trace"
	"github.com/mtrace-project/mtrace/trigger"
)

func TestTraceToProto_Nil(t *testing.T) {
	var tr *trace.Trace
	proto := tr.ToProto()
	if proto != nil {
		t.Errorf("Expected nil proto for nil trace, got %v", proto)
	}
}

func TestTraceToProto_Full(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")
	startTime := time.Unix(100, 0).UTC()
	endTime := time.Unix(200, 0).UTC()
	dur := 100 * time.Second

	tr := &trace.Trace{
		TraceId:    traceID,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   dur,
		SpanCount:  1,
		ErrorCount: 0,
		Spans: []*span.Span{
			{
				SpanId:        "s1",
				ParentId:      "p1",
				ServiceName:   "service-a",
				OperationName: "op-a",
				SpanKind:      "server",
				SpanStatus:    "ok",
				StartTime:     startTime,
				EndTime:       endTime,
				Duration:      dur,
			},
			nil, // Should be ignored in loop
		},
	}

	proto := tr.ToProto()
	if proto == nil {
		t.Fatal("Expected non-nil proto for populated trace")
	}

	if proto.TraceId != "1234567890abcdef1234567890abcdef" {
		t.Errorf("Expected trace ID %q, got %q", "1234567890abcdef1234567890abcdef", proto.TraceId)
	}
	if proto.SpanCount != 1 {
		t.Errorf("Expected SpanCount 1, got %d", proto.SpanCount)
	}
	if proto.ErrorCount != 0 {
		t.Errorf("Expected ErrorCount 0, got %d", proto.ErrorCount)
	}
	if proto.StartTime.AsTime() != startTime {
		t.Errorf("Expected StartTime %v, got %v", startTime, proto.StartTime.AsTime())
	}
	if proto.EndTime.AsTime() != endTime {
		t.Errorf("Expected EndTime %v, got %v", endTime, proto.EndTime.AsTime())
	}
	if proto.Duration.AsDuration() != dur {
		t.Errorf("Expected Duration %v, got %v", dur, proto.Duration.AsDuration())
	}

	// Verify spans mapping
	if len(proto.Spans) != 1 {
		t.Fatalf("Expected 1 mapped span, got %d", len(proto.Spans))
	}
	s := proto.Spans[0]
	if s.SpanId != "s1" || s.ParentId != "p1" || s.ServiceName != "service-a" || s.OperationName != "op-a" || s.SpanKind != "server" || s.SpanStatus != "ok" {
		t.Errorf("Span fields mismatch in proto: %+v", s)
	}
}
