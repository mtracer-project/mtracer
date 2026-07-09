package trigger_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	testutils "github.com/mtrace-project/mtrace/testUtils"
	"github.com/mtrace-project/mtrace/trigger"
)

func TestParseGrpcMethod(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantService string
		wantMethod  string
		wantErr     bool
	}{
		{
			name:        "simple package, service, method",
			input:       "dice.DiceService.RollDice",
			wantService: "dice.DiceService",
			wantMethod:  "RollDice",
			wantErr:     false,
		},
		{
			name:        "multi-token package, service, method",
			input:       "foo.bar.baz.DiceService.RollDice",
			wantService: "foo.bar.baz.DiceService",
			wantMethod:  "RollDice",
			wantErr:     false,
		},
		{
			name:        "service and method, no package",
			input:       "DiceService.RollDice",
			wantService: "DiceService",
			wantMethod:  "RollDice",
			wantErr:     false,
		},
		{
			name:        "no dot (invalid)",
			input:       "RollDice",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
		},
		{
			name:        "empty string (invalid)",
			input:       "",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotService, gotMethod, err := trigger.ParseGrpcMethod(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseGrpcMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if gotService != tt.wantService {
					t.Errorf("parseGrpcMethod() gotService = %q, want %q", gotService, tt.wantService)
				}
				if gotMethod != tt.wantMethod {
					t.Errorf("parseGrpcMethod() gotMethod = %q, want %q", gotMethod, tt.wantMethod)
				}
			}
		})
	}
}

func TestGrpcTrigger_Errors(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{}

	// Missing serverAddress
	dto1 := &parser.TriggerDTO{
		Type: "grpc",
		Args: map[string]any{
			"method": "package.Service.Method",
		},
	}
	_, err := trigger.NewTrigger(dto1, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing serverAddress")
	}

	// Missing method
	dto2 := &parser.TriggerDTO{
		Type: "grpc",
		Args: map[string]any{
			"serverAddress": "localhost:50051",
		},
	}
	_, err = trigger.NewTrigger(dto2, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing method")
	}

	// Invalid method format
	dto3 := &parser.TriggerDTO{
		Type: "grpc",
		Args: map[string]any{
			"serverAddress": "localhost:50051",
			"method":        "invalid_method_format_no_dot",
		},
	}
	_, err = trigger.NewTrigger(dto3, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for invalid method format")
	}

	// Missing descriptorSource
	dto4 := &parser.TriggerDTO{
		Type: "grpc",
		Args: map[string]any{
			"serverAddress": "localhost:50051",
			"method":        "Service.Method",
		},
	}
	_, err = trigger.NewTrigger(dto4, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing descriptorSource")
	}

	// Invalid descriptorSource type
	dto5 := &parser.TriggerDTO{
		Type: "grpc",
		Args: map[string]any{
			"serverAddress": "localhost:50051",
			"method":        "Service.Method",
			"descriptorSource": map[string]any{
				"type": "invalidSourceType",
			},
		},
	}
	_, err = trigger.NewTrigger(dto5, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for unsupported descriptorSource type")
	}

	// Invalid file descriptor source (missing protoPath)
	dto6 := &parser.TriggerDTO{
		Type: "grpc",
		Args: map[string]any{
			"serverAddress": "localhost:50051",
			"method":        "Service.Method",
			"descriptorSource": map[string]any{
				"type": "file",
			},
		},
	}
	_, err = trigger.NewTrigger(dto6, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing protoPath")
	}
}

func TestGrpcTrigger(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workspaceRoot := filepath.Dir(wd)

	// Start mock gRPC server
	responseData := map[string]any{
		"status": "success",
	}
	mockServer := testutils.StartGRPCTargetServer(
		t,
		workspaceRoot,
		"trigger/testGrpc.proto",
		"dice.DiceService",
		responseData,
	)

	// Stub ID generator
	expectedTraceID := "aaaaaaaabbbbbbbbccccccccdddddddd"
	expectedSpanID := "aaaaaaaabbbbbbbb"
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: expectedTraceID,
		SpanID:  expectedSpanID,
	}

	dto := &parser.TriggerDTO{
		Type: "grpc",
		Args: map[string]any{
			"serverAddress": mockServer.Address,
			"method":        "dice.DiceService.RollDice",
			"descriptorSource": map[string]any{
				"type":      "file",
				"protoPath": "trigger/testGrpc.proto",
			},
			"metadata": map[string]any{
				"authorization": "Bearer token123",
			},
			"data": map[string]any{
				"rollerName": "Bob",
			},
		},
	}

	trig, err := trigger.NewTrigger(dto, mockIDGen, workspaceRoot, context.Background())
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

	// Verify request received by mock gRPC server
	if len(mockServer.Requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(mockServer.Requests))
	}

	req := mockServer.Requests[0]
	if req.Method != "/dice.DiceService/RollDice" {
		t.Errorf("Expected method %q, got %q", "/dice.DiceService/RollDice", req.Method)
	}

	// Check headers/metadata
	authHeaders := req.Metadata.Get("authorization")
	if len(authHeaders) == 0 || authHeaders[0] != "Bearer token123" {
		t.Errorf("Expected authorization header 'Bearer token123', got %v", authHeaders)
	}

	traceparentHeaders := req.Metadata.Get("traceparent")
	expectedTraceparent := fmt.Sprintf("00-%s-%s-01", expectedTraceID, expectedSpanID)
	if len(traceparentHeaders) == 0 || traceparentHeaders[0] != expectedTraceparent {
		t.Errorf("Expected traceparent header %q, got %v", expectedTraceparent, traceparentHeaders)
	}

	// Check request fields
	rollerNameField := req.Request.ProtoReflect().Descriptor().Fields().ByName("rollerName")
	rollerNameVal := req.Request.ProtoReflect().Get(rollerNameField).String()
	if rollerNameVal != "Bob" {
		t.Errorf("Expected rollerName %q, got %q", "Bob", rollerNameVal)
	}
}
