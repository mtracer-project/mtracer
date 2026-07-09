package trace

import (
	"fmt"
	"time"

	"github.com/mtrace-project/mtrace/parser"
	"github.com/mtrace-project/mtrace/span"
	"github.com/mtrace-project/mtrace/trigger"
)

// Trace represents a collection of spans defined with OpenTelemetry conventions
type Trace struct {
	TraceId    trigger.TraceId
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	SpanCount  int
	ErrorCount int
	Spans      []*span.Span
}

func (t *Trace) String() string {
	return fmt.Sprintf("Trace{TraceId: %s, StartTime: %s, EndTime: %s, Duration: %s, SpanCount: %d, ErrorCount: %d, Spans: %v}",
		t.TraceId.String(), t.StartTime, t.EndTime, t.Duration, t.SpanCount, t.ErrorCount, t.Spans)
}

func NewTrace(traceId trigger.TraceId, startTime time.Time, endTime time.Time, duration time.Duration, spanCount int, errorCount int, spans []*span.Span) *Trace {
	return &Trace{
		TraceId:    traceId,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   duration,
		SpanCount:  spanCount,
		ErrorCount: errorCount,
		Spans:      spans,
	}
}

func (t *Trace) GetLastSpan() *span.Span {
	if len(t.Spans) == 0 {
		return nil
	}
	return t.Spans[len(t.Spans)-1]
}

type ExpectedTrace struct {
	spans      []*span.ExpectedSpan
	comparator TraceSpansComparator
}

func (e *ExpectedTrace) String() string {
	return fmt.Sprintf("ExpectedTrace{spans: %v", e.spans)
}

type ExpectedTraceProperties struct {
	maxDuration *time.Duration
	minDuration *time.Duration
	spanCount   *int
	errorCount  *int
}

func NewExpectedTraces(dtos []*parser.ExpectedTraceDTO) []*ExpectedTrace {
	expectedTraces := make([]*ExpectedTrace, len(dtos))
	for i, dto := range dtos {
		expectedTraces[i] = NewExpectedTrace(dto)
	}
	return expectedTraces
}

func NewExpectedTrace(dto *parser.ExpectedTraceDTO) *ExpectedTrace {
	spans := make([]*span.ExpectedSpan, len(dto.Spans))
	for i, spanDTO := range dto.Spans {
		spans[i] = span.NewExpectedSpan(spanDTO)
	}

	var ordered bool
	if dto.Ordered != nil {
		ordered = *dto.Ordered
	} else {
		return nil
	}

	var checker string
	if dto.Checker != nil {
		checker = *dto.Checker
	} else {
		return nil
	}

	comparator := NewTraceSpansComparator(checker, ordered)
	if comparator == nil {
		return nil
	}

	return &ExpectedTrace{
		spans:      spans,
		comparator: comparator,
	}
}

func NewExpectedTraceProperties(dto *parser.ExpectedTracePropertiesDTO) *ExpectedTraceProperties {
	if dto == nil {
		return nil
	}
	return &ExpectedTraceProperties{
		maxDuration: dto.MaxDuration.ToTimeDuration(),
		minDuration: dto.MinDuration.ToTimeDuration(),
		spanCount:   dto.SpanCount,
		errorCount:  dto.ErrorCount,
	}
}

// Adapter and Repository
type TraceAdapter interface {
	Fetch(traceId trigger.TraceId, timeout time.Duration, retryDelay time.Duration, lastSpan *span.ExpectedSpan) (*Trace, error)
}

type TraceComparator interface {
	Compare() (bool, error)
}

// Comparator
type TraceSpansComparator interface {
	Compare(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string)
}
