package jaeger

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/span"
	"github.com/mtrace-project/mtrace/trace"
	"github.com/mtrace-project/mtrace/trigger"
)

type JaegerTraceAdapter struct {
	repository IJaegerTraceRepository
}

func (j *JaegerTraceAdapter) Fetch(traceId trigger.TraceId, timeout time.Duration, retryDelay time.Duration, lastSpan *span.ExpectedSpan) (*trace.Trace, error) {
	calledAt := time.Now()
	var tr *trace.Trace

	// retry delay
	ticker := time.NewTicker(retryDelay)
	defer ticker.Stop()

	// timeout
	timeoutTimer := time.NewTimer(time.Until(calledAt.Add(timeout)))
	timeoutCh := timeoutTimer.C
	defer timeoutTimer.Stop()
	timedOut := false

	for !timedOut {
		response, err := j.repository.Get(traceId)
		if err != nil {
			slog.Debug("Error fetching trace, retrying...", "error", err)
		} else {
			tr, err = newTraceFromJaeger(response)
			if err != nil {
				return nil, fmt.Errorf("error converting Jaeger response to trace: %v", err)
			}
			if tr != nil {
				actualLastSpan := tr.GetLastSpan()
				equal, _ := actualLastSpan.Equal(lastSpan)
				if equal {
					return tr, nil
				}

				slog.Debug("Last span not found yet, retrying...", "expected", lastSpan, "actual", actualLastSpan)
			}
		}

		select {
		case <-timeoutCh:
			timedOut = true
		case <-ticker.C:
			// iterate every retryDelay until it finds the exact last span or timeouts
		}
	}

	if tr == nil {
		return nil, fmt.Errorf("trace with ID %s not found within timeout", traceId)
	}

	return tr, nil
}

func NewJaegerTraceAdapter(repository IJaegerTraceRepository) (*JaegerTraceAdapter, error) {
	return &JaegerTraceAdapter{
		repository: repository,
	}, nil
}

func newTraceFromJaeger(response *JaegerTraceDTO) (*trace.Trace, error) {
	spans := make([]*span.Span, len(response.Spans))
	var minStart time.Time
	var maxEnd time.Time
	errorCount := 0

	for i, spanDTO := range response.Spans {
		// ParentId resolution
		parentId := ""
		for _, ref := range spanDTO.References {
			if ref.RefType == "CHILD_OF" {
				parentId = ref.SpanId
				break
			}
		}

		// ServiceName resolution
		serviceName := ""
		if proc, exists := response.Processes[spanDTO.ProcessID]; exists {
			serviceName = proc.ServiceName
		}

		// SpanKind resolution
		spanKind := ""
		for _, tag := range spanDTO.Tags {
			if tag.Key == "span.kind" {
				if strVal, ok := tag.Value.(string); ok {
					spanKind = domain.SpanKindValue(strVal)
				} else {
					spanKind = domain.SpanKindValue(fmt.Sprintf("%v", tag.Value))
				}
				break
			}
		}

		// SpanStatus resolution
		spanStatus := getSpanStatus(spanDTO.Tags)
		if spanStatus == "error" {
			errorCount++
		}

		// Attributes
		attributes := make(map[string]any)
		for _, tag := range spanDTO.Tags {
			attributes[tag.Key] = tag.Value
		}

		// Time resolution
		startTime := time.UnixMicro(spanDTO.StartTimeUs).UTC()
		duration := time.Duration(spanDTO.DurationUs) * time.Microsecond
		endTime := startTime.Add(duration)

		if minStart.IsZero() || startTime.Before(minStart) {
			minStart = startTime
		}
		if maxEnd.IsZero() || endTime.After(maxEnd) {
			maxEnd = endTime
		}

		spans[i] = &span.Span{
			SpanId:        spanDTO.SpanId,
			ParentId:      parentId,
			ServiceName:   serviceName,
			OperationName: spanDTO.OperationName,
			SpanKind:      spanKind,
			SpanStatus:    spanStatus,
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      duration,
			Attributes:    attributes,
		}
	}

	d := maxEnd.Sub(minStart)

	traceId, err := trigger.NewTraceId(response.TraceId)
	if err != nil {
		return nil, fmt.Errorf("invalid trace ID in Jaeger response: %v", err)
	}

	return &trace.Trace{
		TraceId:    traceId,
		StartTime:  minStart,
		EndTime:    maxEnd,
		Duration:   d,
		SpanCount:  len(response.Spans),
		ErrorCount: errorCount,
		Spans:      trace.SortSpansHierarchically(spans),
	}, nil
}

// to verify span status, because Jaeger doesn't have a standard way to represent span status
func getSpanStatus(tags []JaegerTag) string {
	status := "unset"
	for _, tag := range tags {
		if tag.Key == "otel.status_code" {
			status = fmt.Sprintf("%s", tag.Value)
			break
		}
		if tag.Key == "error" {
			status = "error"
			break
		}
	}

	switch strings.ToLower(status) {
	case "ok", "error", "unset":
		// valid
	default:
		status = "unset"
	}

	return status
}
