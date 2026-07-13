package openobserve

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/span"
	"github.com/mtracer-project/mtracer/trace"
	"github.com/mtracer-project/mtracer/trigger"
)

type OpenObserveTraceAdapter struct {
	repository IOpenObserveTraceRepository
}

func (o *OpenObserveTraceAdapter) Fetch(traceId trigger.TraceId, timeout time.Duration, retryDelay time.Duration, lastSpan *span.ExpectedSpan) (*trace.Trace, error) {
	calledAt := time.Now()
	var trace *trace.Trace

	// retry delay
	ticker := time.NewTicker(retryDelay)
	defer ticker.Stop()

	// timeout
	timeoutTimer := time.NewTimer(time.Until(calledAt.Add(timeout)))
	timeoutCh := timeoutTimer.C
	defer timeoutTimer.Stop()
	timedOut := false

	for !timedOut {
		response, err := o.repository.Get(traceId)
		if err != nil {
			slog.Debug("Error fetching trace, retrying...", "error", err)
		} else {
			trace = newTraceFromOpenObserve(response)
			actualLastSpan := trace.GetLastSpan()
			equal, _ := actualLastSpan.Equal(lastSpan)
			if equal {
				return trace, nil
			}

			slog.Debug("Last span not found yet, retrying...", "expected", lastSpan, "actual", actualLastSpan)
		}

		select {
		case <-timeoutCh:
			timedOut = true
		case <-ticker.C:
			// iterate every retryDelay until it finds the exact last span or timeouts
		}
	}

	if trace == nil {
		return nil, fmt.Errorf("trace with ID %s not found within timeout", traceId)
	}

	return trace, nil
}

func NewOpenObserveTraceAdapter(repository IOpenObserveTraceRepository) (*OpenObserveTraceAdapter, error) {
	return &OpenObserveTraceAdapter{
		repository: repository,
	}, nil
}

func newTraceFromOpenObserve(response *OpenObserveTraceResponse) *trace.Trace {
	if response == nil {
		return nil
	}

	var minStart time.Time
	var maxEnd time.Time

	spans := make([]*span.Span, len(response.Spans))
	for i, spanDTO := range response.Spans {
		duration := time.Duration(spanDTO.DurationNs) * time.Nanosecond
		startTime := domain.NanosecondsToTime(spanDTO.StartTimeNs)
		endTime := domain.NanosecondsToTime(spanDTO.EndTimeNs)

		if minStart.IsZero() || startTime.Before(minStart) {
			minStart = startTime
		}
		if maxEnd.IsZero() || endTime.After(maxEnd) {
			maxEnd = endTime
		}

		spans[i] = &span.Span{
			SpanId:        spanDTO.SpanId,
			ParentId:      spanDTO.ParentId,
			ServiceName:   spanDTO.ServiceName,
			OperationName: domain.DerefString(spanDTO.OperationName),
			SpanKind:      domain.DerefString(spanDTO.SpanKind),
			SpanStatus:    domain.DerefString(spanDTO.SpanStatus),
			StartTime:     startTime,
			EndTime:       endTime,
			Duration:      duration,
			Attributes:    spanDTO.Attributes,
		}
	}

	duration := maxEnd.Sub(minStart)

	return &trace.Trace{
		TraceId:    response.TraceId,
		StartTime:  minStart,
		EndTime:    maxEnd,
		Duration:   duration,
		SpanCount:  response.SpanCount,
		ErrorCount: response.ErrorCount,
		Spans:      trace.SortSpansHierarchically(spans),
	}
}
