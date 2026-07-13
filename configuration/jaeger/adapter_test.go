package jaeger_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/configuration/jaeger"
	"github.com/mtracer-project/mtracer/parser"
	"github.com/mtracer-project/mtracer/span"
	"github.com/mtracer-project/mtracer/trigger"
)

type mockJaegerTraceRepository struct {
	GetFunc func(traceId trigger.TraceId) (*jaeger.JaegerTraceDTO, error)
	calls   int
}

func (m *mockJaegerTraceRepository) Get(traceId trigger.TraceId) (*jaeger.JaegerTraceDTO, error) {
	m.calls++
	return m.GetFunc(traceId)
}

func TestJaegerTraceAdapter_Fetch_SuccessFirstTry(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")

	opName := "op-test"
	kind := "server"
	status := "unset"

	expectedLastSpanDTO := &parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName:   "service-test",
			OperationName: &opName,
			SpanKind:      &kind,
			SpanStatus:    &status,
		},
	}
	expectedLastSpan := span.NewExpectedSpan(expectedLastSpanDTO)

	mockRepo := &mockJaegerTraceRepository{
		GetFunc: func(id trigger.TraceId) (*jaeger.JaegerTraceDTO, error) {
			return &jaeger.JaegerTraceDTO{
				TraceId: id.String(),
				Spans: []jaeger.JaegerSpanDTO{
					{
						SpanId:        "s1",
						OperationName: "op-test",
						StartTimeUs:   1000,
						DurationUs:    1000,
						ProcessID:     "p1",
						Tags: []jaeger.JaegerTag{
							{Key: "span.kind", Type: "string", Value: "server"},
						},
					},
				},
				Processes: map[string]jaeger.JaegerProcess{
					"p1": {ServiceName: "service-test"},
				},
			}, nil
		},
	}

	adapter, err := jaeger.NewJaegerTraceAdapter(mockRepo)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	result, err := adapter.Fetch(traceID, 100*time.Millisecond, 10*time.Millisecond, expectedLastSpan)
	if err != nil {
		t.Fatalf("Unexpected error fetching trace: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil trace result")
	}
	if mockRepo.calls != 1 {
		t.Errorf("Expected 1 call to repository, got %d", mockRepo.calls)
	}
	if result.TraceId != traceID {
		t.Errorf("Expected trace ID %q, got %q", traceID, result.TraceId)
	}
}

func TestJaegerTraceAdapter_Fetch_SuccessAfterRetries(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")

	opName := "op-test"
	kind := "server"
	status := "unset"

	expectedLastSpan := span.NewExpectedSpan(&parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName:   "service-test",
			OperationName: &opName,
			SpanKind:      &kind,
			SpanStatus:    &status,
		},
	})

	callCount := 0
	mockRepo := &mockJaegerTraceRepository{
		GetFunc: func(id trigger.TraceId) (*jaeger.JaegerTraceDTO, error) {
			callCount++
			if callCount == 1 {
				// Return mismatching last span (e.g. ServiceName = "wrong-service")
				return &jaeger.JaegerTraceDTO{
					TraceId: id.String(),
					Spans: []jaeger.JaegerSpanDTO{
						{
							SpanId:        "s1",
							OperationName: "wrong-op",
							StartTimeUs:   1000,
							DurationUs:    1000,
							ProcessID:     "p1",
							Tags: []jaeger.JaegerTag{
								{Key: "span.kind", Type: "string", Value: "client"},
								{Key: "error", Type: "bool", Value: true},
							},
						},
					},
					Processes: map[string]jaeger.JaegerProcess{
						"p1": {ServiceName: "wrong-service"},
					},
				}, nil
			}

			// Return matching span on call 2
			return &jaeger.JaegerTraceDTO{
				TraceId: id.String(),
				Spans: []jaeger.JaegerSpanDTO{
					{
						SpanId:        "s1",
						OperationName: "op-test",
						StartTimeUs:   1000,
						DurationUs:    1000,
						ProcessID:     "p1",
						Tags: []jaeger.JaegerTag{
							{Key: "span.kind", Type: "string", Value: "server"},
						},
					},
				},
				Processes: map[string]jaeger.JaegerProcess{
					"p1": {ServiceName: "service-test"},
				},
			}, nil
		},
	}

	adapter, err := jaeger.NewJaegerTraceAdapter(mockRepo)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	result, err := adapter.Fetch(traceID, 200*time.Millisecond, 10*time.Millisecond, expectedLastSpan)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil trace result")
	}
	if callCount != 2 {
		t.Errorf("Expected exactly 2 repository calls, got %d", callCount)
	}
	if result.Spans[0].ServiceName != "service-test" {
		t.Errorf("Expected matching last span ServiceName %q, got %q", "service-test", result.Spans[0].ServiceName)
	}
}

func TestJaegerTraceAdapter_Fetch_TimeoutUnmatchedLastSpan(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")

	opName := "op-test"
	kind := "server"
	status := "unset"

	expectedLastSpan := span.NewExpectedSpan(&parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName:   "service-test",
			OperationName: &opName,
			SpanKind:      &kind,
			SpanStatus:    &status,
		},
	})

	callCount := 0
	mockRepo := &mockJaegerTraceRepository{
		GetFunc: func(id trigger.TraceId) (*jaeger.JaegerTraceDTO, error) {
			callCount++
			// Always return wrong span
			return &jaeger.JaegerTraceDTO{
				TraceId: id.String(),
				Spans: []jaeger.JaegerSpanDTO{
					{
						SpanId:        "s1",
						OperationName: "wrong-op",
						StartTimeUs:   1000,
						DurationUs:    1000,
						ProcessID:     "p1",
					},
				},
				Processes: map[string]jaeger.JaegerProcess{
					"p1": {ServiceName: "wrong-service"},
				},
			}, nil
		},
	}

	adapter, err := jaeger.NewJaegerTraceAdapter(mockRepo)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Fetch with a short timeout
	result, err := adapter.Fetch(traceID, 50*time.Millisecond, 10*time.Millisecond, expectedLastSpan)
	if err != nil {
		t.Fatalf("Expected nil error when trace is found (even if unmatched last span), got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil trace result")
	}

	if callCount <= 1 {
		t.Errorf("Expected multiple repository retries before timeout, got %d", callCount)
	}
}

func TestJaegerTraceAdapter_Fetch_TimeoutRepositoryErrors(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")

	callCount := 0
	mockRepo := &mockJaegerTraceRepository{
		GetFunc: func(id trigger.TraceId) (*jaeger.JaegerTraceDTO, error) {
			callCount++
			return nil, fmt.Errorf("repository error")
		},
	}

	adapter, err := jaeger.NewJaegerTraceAdapter(mockRepo)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	_, err = adapter.Fetch(traceID, 50*time.Millisecond, 10*time.Millisecond, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if callCount <= 1 {
		t.Errorf("Expected multiple repository retries before timeout, got %d", callCount)
	}
}

func TestJaegerTraceAdapter_Fetch_RepositoryErrorThenSuccess(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")

	opName := "op-test"
	kind := "server"
	status := "unset"

	expectedLastSpan := span.NewExpectedSpan(&parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName:   "service-test",
			OperationName: &opName,
			SpanKind:      &kind,
			SpanStatus:    &status,
		},
	})

	callCount := 0
	mockRepo := &mockJaegerTraceRepository{
		GetFunc: func(id trigger.TraceId) (*jaeger.JaegerTraceDTO, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("temporary repository error")
			}
			return &jaeger.JaegerTraceDTO{
				TraceId: id.String(),
				Spans: []jaeger.JaegerSpanDTO{
					{
						SpanId:        "s1",
						OperationName: "op-test",
						StartTimeUs:   1000,
						DurationUs:    1000,
						ProcessID:     "p1",
						Tags: []jaeger.JaegerTag{
							{Key: "span.kind", Type: "string", Value: "server"},
						},
					},
				},
				Processes: map[string]jaeger.JaegerProcess{
					"p1": {ServiceName: "service-test"},
				},
			}, nil
		},
	}

	adapter, err := jaeger.NewJaegerTraceAdapter(mockRepo)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	result, err := adapter.Fetch(traceID, 200*time.Millisecond, 10*time.Millisecond, expectedLastSpan)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil trace result")
	}
	if callCount != 2 {
		t.Errorf("Expected exactly 2 repository calls, got %d", callCount)
	}
}

func TestJaegerTraceAdapter_Fetch_AttributesFromTags(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")

	opName := "op-attr"
	kind := "server"
	status := "unset"

	expectedLastSpan := span.NewExpectedSpan(&parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName:   "attr-service",
			OperationName: &opName,
			SpanKind:      &kind,
			SpanStatus:    &status,
		},
	})

	mockRepo := &mockJaegerTraceRepository{
		GetFunc: func(id trigger.TraceId) (*jaeger.JaegerTraceDTO, error) {
			return &jaeger.JaegerTraceDTO{
				TraceId: id.String(),
				Spans: []jaeger.JaegerSpanDTO{
					{
						SpanId:        "s1",
						OperationName: "op-attr",
						StartTimeUs:   1000,
						DurationUs:    1000,
						ProcessID:     "p1",
						Tags: []jaeger.JaegerTag{
							{Key: "span.kind", Type: "string", Value: "server"},
							{Key: "http.method", Type: "string", Value: "GET"},
							{Key: "http.status_code", Type: "int64", Value: float64(200)},
							{Key: "db.system", Type: "string", Value: "postgresql"},
						},
					},
				},
				Processes: map[string]jaeger.JaegerProcess{
					"p1": {ServiceName: "attr-service"},
				},
			}, nil
		},
	}

	adapter, err := jaeger.NewJaegerTraceAdapter(mockRepo)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	result, err := adapter.Fetch(traceID, 100*time.Millisecond, 10*time.Millisecond, expectedLastSpan)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil || len(result.Spans) == 0 {
		t.Fatal("Expected non-nil trace with spans")
	}

	attrs := result.Spans[0].Attributes
	if attrs == nil {
		t.Fatal("Expected non-nil Attributes on span")
	}

	if attrs["http.method"] != "GET" {
		t.Errorf("Expected attribute http.method 'GET', got %v", attrs["http.method"])
	}
	if attrs["http.status_code"] != float64(200) {
		t.Errorf("Expected attribute http.status_code 200, got %v", attrs["http.status_code"])
	}
	if attrs["db.system"] != "postgresql" {
		t.Errorf("Expected attribute db.system 'postgresql', got %v", attrs["db.system"])
	}
	// span.kind should also be in attributes since all tags are mapped
	if attrs["span.kind"] != "server" {
		t.Errorf("Expected attribute span.kind 'server', got %v", attrs["span.kind"])
	}
}
