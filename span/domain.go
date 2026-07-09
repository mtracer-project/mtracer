package span

import (
	"fmt"
	"time"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/parser"
)

type Span struct {
	SpanId        string
	ParentId      string
	ServiceName   string
	OperationName string
	SpanKind      string
	SpanStatus    string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Attributes    map[string]any
}

func (s *Span) String() string {
	return fmt.Sprintf("Span{SpanId: %s, ServiceName: %s, OperationName: %s, SpanKind: %s, SpanStatus: %s, StartTime: %s, EndTime: %s, Duration: %s, ParentId: %s, Attributes: %v}", s.SpanId, s.ServiceName, s.OperationName, s.SpanKind, s.SpanStatus, s.StartTime, s.EndTime, s.Duration, s.ParentId, s.Attributes)
}

func NewSpan(
	spanId *string,
	parentId *string,
	serviceName *string,
	operationName *string,
	spanKind *string,
	spanStatus *string,
	startTime *time.Time,
	endTime *time.Time,
	duration *time.Duration,
	attributes map[string]any,
) *Span {
	return &Span{
		SpanId:        *spanId,
		ParentId:      *parentId,
		ServiceName:   *serviceName,
		OperationName: *operationName,
		SpanKind:      *spanKind,
		SpanStatus:    *spanStatus,
		StartTime:     *startTime,
		EndTime:       *endTime,
		Duration:      *duration,
		Attributes:    attributes,
	}
}

type ExpectedSpan struct {
	ServiceName   string
	OperationName *string
	SpanKind      *string
	SpanStatus    *string

	maxDuration *time.Duration
	minDuration *time.Duration
}

func (e *ExpectedSpan) String() string {
	return fmt.Sprintf("ExpectedSpan{ServiceName: %s, OperationName: %s, SpanKind: %s, SpanStatus: %s, maxDuration: %s, minDuration: %s}",
		e.ServiceName, domain.DerefString(e.OperationName), domain.DerefString(e.SpanKind), domain.DerefString(e.SpanStatus), e.maxDuration, e.minDuration)
}

func NewExpectedSpan(
	dto *parser.ExpectedSpanDTO,
) *ExpectedSpan {
	if dto == nil {
		return nil
	}

	return &ExpectedSpan{
		ServiceName:   dto.ServiceName,
		OperationName: dto.OperationName,
		SpanKind:      dto.SpanKind,
		SpanStatus:    dto.SpanStatus,
		maxDuration:   dto.MaxDuration.ToTimeDuration(),
		minDuration:   dto.MinDuration.ToTimeDuration(),
	}
}
