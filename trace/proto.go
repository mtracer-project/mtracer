package trace

import (
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mtrace-project/mtrace/span"
)

func (t *Trace) ToProto() *TraceProto {
	if t == nil {
		return nil
	}

	protoTrace := &TraceProto{
		TraceId:    t.TraceId.String(),
		SpanCount:  int32(t.SpanCount),
		ErrorCount: int32(t.ErrorCount),
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
