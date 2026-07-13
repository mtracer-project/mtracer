package trigger_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	testutils "github.com/mtracer-project/mtracer/testUtils"
	"github.com/mtracer-project/mtracer/trigger"
)

func TestHTTPTrigger(t *testing.T) {
	// Start mock HTTP server
	mockResponse := `{"status": "ok"}`
	mockServer := testutils.StartHTTPTargetServer(t, mockResponse, http.StatusOK)

	// Stub ID generator
	expectedTraceID := "1234567890abcdef1234567890abcdef"
	expectedSpanID := "1234567890abcdef"
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: expectedTraceID,
		SpanID:  expectedSpanID,
	}

	// Create HTTPTrigger DTO
	dto := &parser.TriggerDTO{
		Type: "http",
		Args: map[string]any{
			"url":    mockServer.Server.URL,
			"method": "POST",
			"headers": map[string]any{
				"Custom-Header": "Custom-Value",
			},
			"body": `{"foo": "bar"}`,
		},
	}

	ctx := context.Background()
	trig, err := trigger.NewTrigger(dto, mockIDGen, "", ctx)
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	traceID, err := trig.Trigger()
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	if traceID.String() != expectedTraceID {
		t.Errorf("Expected trace ID %q, got %q", expectedTraceID, traceID.String())
	}

	// Verify request received by mock server
	if len(mockServer.Requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(mockServer.Requests))
	}

	req := mockServer.Requests[0]
	if req.Method != "POST" {
		t.Errorf("Expected POST method, got %q", req.Method)
	}
	if req.Body != `{"foo": "bar"}` {
		t.Errorf("Expected body %q, got %q", `{"foo": "bar"}`, req.Body)
	}

	// Check headers
	customVal := req.Headers.Get("Custom-Header")
	if customVal != "Custom-Value" {
		t.Errorf("Expected Custom-Header %q, got %q", "Custom-Value", customVal)
	}

	traceparent := req.Headers.Get("traceparent")
	expectedTraceparent := fmt.Sprintf("00-%s-%s-01", expectedTraceID, expectedSpanID)
	if traceparent != expectedTraceparent {
		t.Errorf("Expected traceparent %q, got %q", expectedTraceparent, traceparent)
	}
}

func TestHTTPTrigger_Errors(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{}

	// Missing URL
	dto1 := &parser.TriggerDTO{
		Type: "http",
		Args: map[string]any{
			"method": "GET",
		},
	}
	_, err := trigger.NewTrigger(dto1, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing url argument")
	}

	// Missing Args entirely
	dto2 := &parser.TriggerDTO{
		Type: "http",
	}
	_, err = trigger.NewTrigger(dto2, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing args")
	}

	// Trigger execution error (e.g. invalid protocol / schema in URL)
	dto3 := &parser.TriggerDTO{
		Type: "http",
		Args: map[string]any{
			"url": "invalid://some-bad-url-address",
		},
	}
	trig, err := trigger.NewTrigger(dto3, mockIDGen, "", context.Background())
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}
	_, err = trig.Trigger()
	if err == nil {
		t.Error("Expected error during trigger call with invalid URL scheme")
	}
}
