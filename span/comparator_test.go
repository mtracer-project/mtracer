package span_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/parser"
	"github.com/mtracer-project/mtracer/span"
)

func TestSpanEqual_NilInputs(t *testing.T) {
	// Case 1: nil expected span
	s := &span.Span{}
	equal, reason := s.Equal(nil)
	if equal || reason != "expected span is nil" {
		t.Errorf("Expected failure for nil expected span, got equal=%v, reason=%q", equal, reason)
	}

	// Case 2: nil actual span
	var nilSpan *span.Span
	expected := &span.ExpectedSpan{}
	equal, reason = nilSpan.Equal(expected)
	if equal || reason != "actual span is nil" {
		t.Errorf("Expected failure for nil actual span, got equal=%v, reason=%q", equal, reason)
	}
}

func TestSpanEqual_FieldMatching(t *testing.T) {
	op := "op-a"
	kind := "server"
	status := "ok"

	expected := span.NewExpectedSpan(&parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName:   "Service-A",
			OperationName: &op,
			SpanKind:      &kind,
			SpanStatus:    &status,
		},
	})

	// Case 1: exact match
	sMatch := &span.Span{
		ServiceName:   "Service-A",
		OperationName: "op-a",
		SpanKind:      "server",
		SpanStatus:    "ok",
	}
	equal, reason := sMatch.Equal(expected)
	if !equal {
		t.Errorf("Expected spans to be equal, got reason: %s", reason)
	}

	// Case 2: case-insensitive match
	sFoldMatch := &span.Span{
		ServiceName:   "service-a",
		OperationName: "OP-A",
		SpanKind:      "SERVER",
		SpanStatus:    "OK",
	}
	equal, reason = sFoldMatch.Equal(expected)
	if !equal {
		t.Errorf("Expected case-insensitive spans to be equal, got reason: %s", reason)
	}

	// Case 3: ServiceName mismatch
	sBadService := &span.Span{
		ServiceName:   "Service-B",
		OperationName: "op-a",
		SpanKind:      "server",
		SpanStatus:    "ok",
	}
	equal, reason = sBadService.Equal(expected)
	if equal || !strings.Contains(reason, "service name does not match") {
		t.Errorf("Expected ServiceName mismatch, got equal=%v, reason=%q", equal, reason)
	}

	// Case 4: OperationName mismatch
	sBadOp := &span.Span{
		ServiceName:   "Service-A",
		OperationName: "op-b",
		SpanKind:      "server",
		SpanStatus:    "ok",
	}
	equal, reason = sBadOp.Equal(expected)
	if equal || !strings.Contains(reason, "operation name does not match") {
		t.Errorf("Expected OperationName mismatch, got equal=%v, reason=%q", equal, reason)
	}

	// Case 5: SpanKind mismatch
	sBadKind := &span.Span{
		ServiceName:   "Service-A",
		OperationName: "op-a",
		SpanKind:      "client",
		SpanStatus:    "ok",
	}
	equal, reason = sBadKind.Equal(expected)
	if equal || !strings.Contains(reason, "span kind does not match") {
		t.Errorf("Expected SpanKind mismatch, got equal=%v, reason=%q", equal, reason)
	}

	// Case 6: SpanStatus mismatch
	sBadStatus := &span.Span{
		ServiceName:   "Service-A",
		OperationName: "op-a",
		SpanKind:      "server",
		SpanStatus:    "error",
	}
	equal, reason = sBadStatus.Equal(expected)
	if equal || !strings.Contains(reason, "span status does not match") {
		t.Errorf("Expected SpanStatus mismatch, got equal=%v, reason=%q", equal, reason)
	}
}

func TestSpanEqual_DurationMatching(t *testing.T) {
	maxDur := domain.Duration(100 * time.Millisecond)
	minDur := domain.Duration(10 * time.Millisecond)

	expected := span.NewExpectedSpan(&parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName: "service",
		},
		MaxDuration: &maxDur,
		MinDuration: &minDur,
	})

	// Case 1: Duration in range
	sMatch := &span.Span{
		ServiceName: "service",
		Duration:    50 * time.Millisecond,
	}
	equal, reason := sMatch.Equal(expected)
	if !equal {
		t.Errorf("Expected duration in range to match, got reason: %s", reason)
	}

	// Case 2: Duration exceeds max
	sExceeds := &span.Span{
		ServiceName: "service",
		Duration:    150 * time.Millisecond,
	}
	equal, reason = sExceeds.Equal(expected)
	if equal || !strings.Contains(reason, "duration exceeds maximum") {
		t.Errorf("Expected duration exceeds max mismatch, got equal=%v, reason=%q", equal, reason)
	}

	// Case 3: Duration below min
	sBelow := &span.Span{
		ServiceName: "service",
		Duration:    5 * time.Millisecond,
	}
	equal, reason = sBelow.Equal(expected)
	if equal || !strings.Contains(reason, "duration is less than minimum") {
		t.Errorf("Expected duration less than min mismatch, got equal=%v, reason=%q", equal, reason)
	}
}
