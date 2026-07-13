package trigger

import (
	"context"
	"fmt"

	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"
)

type TraceIdTrigger struct {
	traceId TraceId
}

func (t *TraceIdTrigger) Trigger() (TraceId, error) {
	return t.traceId, nil
}

func (t *TraceIdTrigger) Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error {
	if dto.Args == nil {
		return fmt.Errorf("invalid trigger arguments")
	}

	traceId, ok := dto.Args["traceId"].(string)
	if !ok {
		return fmt.Errorf("traceId argument is required and must be a string")
	}

	traceIdObj, err := NewTraceId(traceId)
	if err != nil {
		return fmt.Errorf("error while creating TraceId object: %w", err)
	}

	t.traceId = traceIdObj
	return nil
}

func (t *TraceIdTrigger) Example() string {
	return `trigger:
  type: "traceId"
  args:
    traceId: "4bf92f3577b34da6a3ce929d0e0e4736"`
}
