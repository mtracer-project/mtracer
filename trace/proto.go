package trace

import (
	"math"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mtracer-project/mtracer/span"
)

func (t *Trace) ToProto() *TraceProto {
	if t == nil {
		return nil
	}

	var spanCount int32
	if t.SpanCount > math.MaxInt32 {
		spanCount = math.MaxInt32
	} else {
		spanCount = int32(t.SpanCount) // nolint:gosec
	}

	var errorCount int32
	if t.ErrorCount > math.MaxInt32 {
		errorCount = math.MaxInt32
	} else {
		errorCount = int32(t.ErrorCount) // nolint:gosec
	}

	protoTrace := &TraceProto{
		TraceId:    t.TraceId.String(),
		SpanCount:  spanCount,
		ErrorCount: errorCount,
	}

	protoTrace.StartTime = timestamppb.New(t.StartTime)
	protoTrace.EndTime = timestamppb.New(t.EndTime)
	protoTrace.Duration = durationpb.New(t.Duration)

	if len(t.Spans) > 0 {
		protoTrace.Spans = make([]*span.SpanProto, 0, len(t.Spans))
		for _, s := range t.Spans {
			if s == nil {
				continue
			}
			protoTrace.Spans = append(protoTrace.Spans, s.ToProto())
		}
	}

	return protoTrace
}
