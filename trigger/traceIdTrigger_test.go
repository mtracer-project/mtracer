package trigger_test

import (
	"context"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	testutils "github.com/mtracer-project/mtracer/testUtils"
	"github.com/mtracer-project/mtracer/trigger"
)

func TestTraceIdTrigger(t *testing.T) {
	// Case 1: Valid trace ID
	validTraceID := "abcdefabcdefabcdefabcdefabcdef12"
	dto := &parser.TriggerDTO{
		Type: "traceId",
		Args: map[string]any{
			"traceId": validTraceID,
		},
	}

	mockIDGen := &testutils.MockIdGenerator{}
	trig, err := trigger.NewTrigger(dto, mockIDGen, "", context.Background())
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	traceID, err := trig.Trigger()
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	if traceID.String() != validTraceID {
		t.Errorf("Expected trace ID %q, got %q", validTraceID, traceID.String())
	}

	// Case 2: Missing trace ID argument
	invalidDto := &parser.TriggerDTO{
		Type: "traceId",
		Args: map[string]any{},
	}
	_, err = trigger.NewTrigger(invalidDto, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing traceId argument")
	}

	// Case 3: Invalid trace ID format
	badTraceIdDto := &parser.TriggerDTO{
		Type: "traceId",
		Args: map[string]any{
			"traceId": "invalid-hex-id-too-short",
		},
	}
	_, err = trigger.NewTrigger(badTraceIdDto, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for invalid traceId format")
	}

	allZerosTraceIdDto := &parser.TriggerDTO{
		Type: "traceId",
		Args: map[string]any{
			"traceId": "00000000000000000000000000000000",
		},
	}
	_, err = trigger.NewTrigger(allZerosTraceIdDto, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for all zeros traceId")
	}
}
