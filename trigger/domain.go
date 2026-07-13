package trigger

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"
)

type TraceId string

func (t TraceId) String() string {
	return string(t)
}

func NewTraceId(traceId string) (TraceId, error) {
	if traceId == "" {
		return "", fmt.Errorf("traceId cannot be empty")
	}

	if len(traceId) != idgenerator.TRACE_ID_LENGTH {
		return "", fmt.Errorf("traceId must be %d characters long", idgenerator.TRACE_ID_LENGTH)
	}

	if matched := regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(traceId); !matched {
		return "", fmt.Errorf("expected lowercase hex trace ID, got %q", traceId)
	}

	if traceId == "00000000000000000000000000000000" {
		return "", fmt.Errorf("traceId cannot be all zeros")
	}

	traceIdObj := TraceId(traceId)
	return traceIdObj, nil
}

type Trigger interface {
	Trigger() (TraceId, error)
	Example() string
	Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error
}

func NewTrigger(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) (Trigger, error) {
	t, err := NewTriggerFromType(dto.Type)
	if err != nil {
		return nil, err
	}

	err = t.Init(dto, idGenerator, baseDir, ctx)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func NewTriggerFromType(triggerType string) (Trigger, error) {
	switch strings.ToLower(triggerType) {
	case "http":
		return &HTTPTrigger{}, nil
	case "traceid":
		return &TraceIdTrigger{}, nil
	case "nats":
		return &NATSTrigger{}, nil
	case "jetstream":
		return &JetstreamTrigger{}, nil
	case "grpc":
		return &GrpcTrigger{}, nil
	case "playwright":
		return &PlaywrightTrigger{}, nil
	default:
		return nil, fmt.Errorf("unsupported trigger type: %s", triggerType)
	}
}

func getTraceparent(traceId string, spanId string) string {
	return fmt.Sprintf("00-%s-%s-01", traceId, spanId) // version 00, traceId, spanId, sampled 01 (means that spans are going to be stored in the obs. backend)
}

func resolvePath(baseDir string, path string) string {
	if path == "" || baseDir == "" || filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(baseDir, path)
}
