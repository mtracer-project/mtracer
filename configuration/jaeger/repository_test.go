package jaeger_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/configuration/jaeger"
	testutils "github.com/mtrace-project/mtrace/testUtils"
	"github.com/mtrace-project/mtrace/trigger"
)

func TestJaegerTraceRepository_Get_Success(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	mockSpans := []jaeger.JaegerSpanDTO{
		{
			TraceId:       traceID.String(),
			SpanId:        "s1",
			OperationName: "op-a",
			References:    nil,
			StartTimeUs:   1700000000000000,
			DurationUs:    5000000,
			Tags: []jaeger.JaegerTag{
				{Key: "span.kind", Type: "string", Value: "server"},
				{Key: "otel.status_code", Type: "string", Value: "UNSET"},
			},
			ProcessID: "p1",
		},
		{
			TraceId:       traceID.String(),
			SpanId:        "s2",
			OperationName: "op-b",
			References: []jaeger.JaegerReference{
				{RefType: "CHILD_OF", TraceId: traceID.String(), SpanId: "s1"},
			},
			StartTimeUs: 1700000001000000,
			DurationUs:  3000000,
			Tags: []jaeger.JaegerTag{
				{Key: "span.kind", Type: "string", Value: "client"},
				{Key: "error", Type: "bool", Value: true},
			},
			ProcessID: "p2",
		},
	}

	mockProcesses := map[string]jaeger.JaegerProcess{
		"p1": {ServiceName: "service-a"},
		"p2": {ServiceName: "service-b"},
	}

	mockTrace := []jaeger.JaegerTraceDTO{
		{
			TraceId:   traceID.String(),
			Spans:     mockSpans,
			Processes: mockProcesses,
		},
	}

	mockServer := testutils.StartMockJaegerServer(t, func(tid string) (any, int) {
		if tid != traceID.String() {
			t.Errorf("Expected trace ID %q, got %q", traceID.String(), tid)
		}
		return mockTrace, http.StatusOK
	})

	repo := jaeger.NewJaegerTraceRepository(
		&jaeger.JaegerConfig{
			BaseURL: mockServer.URL,
		},
		context.Background(),
	)

	resp, err := repo.Get(traceID)
	if err != nil {
		t.Fatalf("Unexpected error fetching trace: %v", err)
	}

	if resp.TraceId != traceID.String() {
		t.Errorf("Expected trace ID %q, got %q", traceID.String(), resp.TraceId)
	}

	if len(resp.Spans) != 2 {
		t.Fatalf("Expected 2 spans, got %d", len(resp.Spans))
	}

	s1 := resp.Spans[0]
	if s1.SpanId != "s1" || s1.OperationName != "op-a" || s1.ProcessID != "p1" {
		t.Errorf("Span 1 field mismatch: %+v", s1)
	}

	s2 := resp.Spans[1]
	if s2.SpanId != "s2" || s2.OperationName != "op-b" || len(s2.References) != 1 || s2.References[0].SpanId != "s1" {
		t.Errorf("Span 2 field mismatch: %+v", s2)
	}

	p1, exists := resp.Processes["p1"]
	if !exists || p1.ServiceName != "service-a" {
		t.Errorf("Process p1 mismatch: %+v", p1)
	}
}

func TestJaegerTraceRepository_Get_EmptyTraceId(t *testing.T) {
	repo := jaeger.NewJaegerTraceRepository(
		&jaeger.JaegerConfig{
			BaseURL: "http://localhost:16686",
		},
		context.Background(),
	)

	_, err := repo.Get("")
	if err == nil {
		t.Error("Expected error when empty trace ID is queried")
	}
}

func TestJaegerTraceRepository_Get_HttpError(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	mockServer := testutils.StartMockJaegerServer(t, func(tid string) (any, int) {
		return nil, http.StatusInternalServerError
	})

	repo := jaeger.NewJaegerTraceRepository(
		&jaeger.JaegerConfig{
			BaseURL: mockServer.URL,
		},
		context.Background(),
	)

	_, err := repo.Get(traceID)
	if err == nil {
		t.Error("Expected error when Jaeger returns 500")
	}
}

func TestJaegerTraceRepository_Get_InvalidJSON(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	server := http.Server{}
	listener, err := net.Listen("tcp", "127.0.0.1:0") // nolint:noctx
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := "http://" + listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid-json"))
	})
	server.Handler = mux

	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
	})

	repo := jaeger.NewJaegerTraceRepository(
		&jaeger.JaegerConfig{
			BaseURL: addr,
		},
		context.Background(),
	)

	_, err = repo.Get(traceID)
	if err == nil {
		t.Error("Expected error when decoding invalid JSON")
	}
}

func TestJaegerTraceRepository_Get_NoHits(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	mockServer := testutils.StartMockJaegerServer(t, func(tid string) (any, int) {
		return []jaeger.JaegerTraceDTO{}, http.StatusOK
	})

	repo := jaeger.NewJaegerTraceRepository(
		&jaeger.JaegerConfig{
			BaseURL: mockServer.URL,
		},
		context.Background(),
	)

	_, err := repo.Get(traceID)
	if err == nil {
		t.Error("Expected error when trace list is empty")
	} else if !strings.Contains(err.Error(), "not found in Jaeger") {
		t.Errorf("Expected 'not found in Jaeger' error, got: %v", err)
	}
}
