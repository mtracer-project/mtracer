package openobserve_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/configuration/openobserve"
	"github.com/mtracer-project/mtracer/parser"
	"github.com/mtracer-project/mtracer/span"
	"github.com/mtracer-project/mtracer/trigger"
)

type mockTraceRepository struct {
	GetFunc func(traceId trigger.TraceId) (*openobserve.OpenObserveTraceResponse, error)
	calls   int
}

func (m *mockTraceRepository) Get(traceId trigger.TraceId) (*openobserve.OpenObserveTraceResponse, error) {
	m.calls++
	return m.GetFunc(traceId)
}

func TestOpenObserveTraceAdapter_Fetch_SuccessFirstTry(t *testing.T) {
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

	mockRepo := &mockTraceRepository{
		GetFunc: func(id trigger.TraceId) (*openobserve.OpenObserveTraceResponse, error) {
			return &openobserve.OpenObserveTraceResponse{
				TraceId:     id,
				StartTimeNs: 1000,
				EndTimeNs:   2000,
				DurationNs:  1000,
				SpanCount:   1,
				ErrorCount:  0,
				Spans: []*openobserve.OpenObserveSpanDTO{
					{
						SpanId:        "s1",
						ServiceName:   "service-test",
						OperationName: &opName,
						SpanKind:      &kind,
						SpanStatus:    &status,
						StartTimeNs:   1000,
						EndTimeNs:     2000,
						DurationNs:    1000,
					},
				},
			}, nil
		},
	}

	adapter, err := openobserve.NewOpenObserveTraceAdapter(mockRepo)
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

func TestOpenObserveTraceAdapter_Fetch_SuccessAfterRetries(t *testing.T) {
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
	mockRepo := &mockTraceRepository{
		GetFunc: func(id trigger.TraceId) (*openobserve.OpenObserveTraceResponse, error) {
			callCount++
			if callCount == 1 {
				// Return mismatching last span (e.g. ServiceName = "wrong-service")
				wrongOp := "wrong-op"
				wrongKind := "client"
				wrongStatus := "error"
				return &openobserve.OpenObserveTraceResponse{
					TraceId:    id,
					SpanCount:  1,
					ErrorCount: 1,
					Spans: []*openobserve.OpenObserveSpanDTO{
						{
							SpanId:        "s1",
							ServiceName:   "wrong-service",
							OperationName: &wrongOp,
							SpanKind:      &wrongKind,
							SpanStatus:    &wrongStatus,
							StartTimeNs:   1000,
							EndTimeNs:     2000,
						},
					},
				}, nil
			}

			// Return matching span on call 2
			return &openobserve.OpenObserveTraceResponse{
				TraceId:    id,
				SpanCount:  1,
				ErrorCount: 0,
				Spans: []*openobserve.OpenObserveSpanDTO{
					{
						SpanId:        "s1",
						ServiceName:   "service-test",
						OperationName: &opName,
						SpanKind:      &kind,
						SpanStatus:    &status,
						StartTimeNs:   1000,
						EndTimeNs:     2000,
					},
				},
			}, nil
		},
	}

	adapter, err := openobserve.NewOpenObserveTraceAdapter(mockRepo)
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

func TestOpenObserveTraceAdapter_Fetch_TimeoutUnmatchedLastSpan(t *testing.T) {
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
	mockRepo := &mockTraceRepository{
		GetFunc: func(id trigger.TraceId) (*openobserve.OpenObserveTraceResponse, error) {
			callCount++
			// Always return wrong span
			wrongOp := "wrong-op"
			return &openobserve.OpenObserveTraceResponse{
				TraceId:   id,
				SpanCount: 1,
				Spans: []*openobserve.OpenObserveSpanDTO{
					{
						SpanId:        "s1",
						ServiceName:   "wrong-service",
						OperationName: &wrongOp,
					},
				},
			}, nil
		},
	}

	adapter, err := openobserve.NewOpenObserveTraceAdapter(mockRepo)
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

func TestOpenObserveTraceAdapter_Fetch_TimeoutRepositoryErrors(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")

	callCount := 0
	mockRepo := &mockTraceRepository{
		GetFunc: func(id trigger.TraceId) (*openobserve.OpenObserveTraceResponse, error) {
			callCount++
			return nil, fmt.Errorf("repository error")
		},
	}

	adapter, err := openobserve.NewOpenObserveTraceAdapter(mockRepo)
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

func TestOpenObserveTraceAdapter_Fetch_RepositoryErrorThenSuccess(t *testing.T) {
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
	mockRepo := &mockTraceRepository{
		GetFunc: func(id trigger.TraceId) (*openobserve.OpenObserveTraceResponse, error) {
			callCount++
			if callCount == 1 {
				return nil, fmt.Errorf("temporary repository error")
			}
			return &openobserve.OpenObserveTraceResponse{
				TraceId:   id,
				SpanCount: 1,
				Spans: []*openobserve.OpenObserveSpanDTO{
					{
						SpanId:        "s1",
						ServiceName:   "service-test",
						OperationName: &opName,
						SpanKind:      &kind,
						SpanStatus:    &status,
						StartTimeNs:   1000,
						EndTimeNs:     2000,
					},
				},
			}, nil
		},
	}

	adapter, err := openobserve.NewOpenObserveTraceAdapter(mockRepo)
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
