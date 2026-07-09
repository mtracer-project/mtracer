package openobserve_test

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/configuration/openobserve"
	testutils "github.com/mtrace-project/mtrace/testUtils"
	"github.com/mtrace-project/mtrace/trigger"
)

func TestOpenObserveTraceRepository_Get_Success(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	// Start mock OpenObserve server
	mockHits := []map[string]any{
		{
			"span_id":                  "s1",
			"reference_parent_span_id": "",
			"service_name":             "service-a",
			"operation_name":           "op-a",
			"span_kind":                "server",
			"span_status":              "unset",
			"start_time":               int64(1700000000000000),
			"end_time":                 int64(1700000005000000),
			"duration":                 int64(5000000),
		},
		{
			"span_id":                  "s2",
			"reference_parent_span_id": "s1",
			"service_name":             "service-b",
			"operation_name":           "op-b",
			"span_kind":                "client",
			"span_status":              "error",
			"start_time":               int64(1700000001000000),
			"end_time":                 int64(1700000004000000),
			"duration":                 int64(3000000),
		},
	}

	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		// Verify query contains trace ID
		if !strings.Contains(sqlQuery, traceID.String()) {
			t.Errorf("Expected query to contain trace ID %q, query: %q", traceID.String(), sqlQuery)
		}
		return mockHits, http.StatusOK
	})

	repo := openobserve.NewOpenObserveTraceRepository(
		&openobserve.OpenObserveConfig{
			BaseURL:    mockServer.URL,
			OrgName:    "org-test",
			StreamName: "stream-test",
			Username:   "user",
			Password:   "pass",
		},
		context.Background(),
	)

	resp, err := repo.Get(traceID)
	if err != nil {
		t.Fatalf("Unexpected error fetching trace: %v", err)
	}

	if resp.TraceId != traceID {
		t.Errorf("Expected trace ID %q, got %q", traceID, resp.TraceId)
	}
	if resp.StartTimeNs != 1700000000000000 {
		t.Errorf("Expected StartTimeNs 1700000000000000, got %d", resp.StartTimeNs)
	}
	if resp.EndTimeNs != 1700000005000000 {
		t.Errorf("Expected EndTimeNs 1700000005000000, got %d", resp.EndTimeNs)
	}
	if resp.DurationNs != 5000000 {
		t.Errorf("Expected DurationNs 5000000, got %d", resp.DurationNs)
	}
	if resp.SpanCount != 2 {
		t.Errorf("Expected SpanCount 2, got %d", resp.SpanCount)
	}
	if resp.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", resp.ErrorCount)
	}

	// Verify spans
	if len(resp.Spans) != 2 {
		t.Fatalf("Expected 2 spans in response, got %d", len(resp.Spans))
	}
	s1 := resp.Spans[0]
	if s1.SpanId != "s1" || s1.ServiceName != "service-a" || *s1.SpanKind != "server" || *s1.SpanStatus != "unset" {
		t.Errorf("Span 1 field mismatch: %+v", s1)
	}
	s2 := resp.Spans[1]
	if s2.SpanId != "s2" || s2.ParentId != "s1" || s2.ServiceName != "service-b" || *s2.SpanKind != "client" || *s2.SpanStatus != "error" {
		t.Errorf("Span 2 field mismatch: %+v", s2)
	}
}

func TestOpenObserveTraceRepository_Get_SpanKinds(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	testCases := []struct {
		inputKind string
		wantKind  string
	}{
		{"0", "unspecified"},
		{"unspecified", "unspecified"},
		{"1", "internal"},
		{"internal", "internal"},
		{"2", "server"},
		{"server", "server"},
		{"3", "client"},
		{"client", "client"},
		{"4", "producer"},
		{"producer", "producer"},
		{"5", "consumer"},
		{"consumer", "consumer"},
		{"custom", "custom"},
		{" CUSTOM  ", "custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.inputKind, func(t *testing.T) {
			mockHits := []map[string]any{
				{
					"span_id":    "s1",
					"span_kind":  tc.inputKind,
					"start_time": int64(1000),
					"end_time":   int64(2000),
				},
			}

			mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
				return mockHits, http.StatusOK
			})

			repo := openobserve.NewOpenObserveTraceRepository(
				&openobserve.OpenObserveConfig{
					BaseURL:    mockServer.URL,
					OrgName:    "org-test",
					StreamName: "stream-test",
					Username:   "user",
					Password:   "pass",
				},
				context.Background(),
			)

			resp, err := repo.Get(traceID)
			if err != nil {
				t.Fatalf("Unexpected error fetching trace: %v", err)
			}

			gotKind := *resp.Spans[0].SpanKind
			if gotKind != tc.wantKind {
				t.Errorf("Expected mapped span kind %q, got %q", tc.wantKind, gotKind)
			}
		})
	}
}

func TestOpenObserveTraceRepository_Get_EmptyTraceId(t *testing.T) {
	repo := openobserve.NewOpenObserveTraceRepository(
		&openobserve.OpenObserveConfig{
			BaseURL:    "http://localhost:5080",
			OrgName:    "org",
			StreamName: "stream",
			Username:   "user",
			Password:   "pass",
		},
		context.Background(),
	)

	_, err := repo.Get("")
	if err == nil {
		t.Error("Expected error when empty trace ID is queried")
	}
}

func TestOpenObserveTraceRepository_Get_HttpError(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		return nil, http.StatusInternalServerError
	})

	repo := openobserve.NewOpenObserveTraceRepository(
		&openobserve.OpenObserveConfig{
			BaseURL:    mockServer.URL,
			OrgName:    "org-test",
			StreamName: "stream-test",
			Username:   "user",
			Password:   "pass",
		},
		context.Background(),
	)

	_, err := repo.Get(traceID)
	if err == nil {
		t.Error("Expected error when OpenObserve returns 500")
	}
}

func TestOpenObserveTraceRepository_Get_InvalidJSON(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	// Create custom server that returns invalid JSON
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

	repo := openobserve.NewOpenObserveTraceRepository(
		&openobserve.OpenObserveConfig{
			BaseURL:    addr,
			OrgName:    "org-test",
			StreamName: "stream-test",
			Username:   "user",
			Password:   "pass",
		},
		context.Background(),
	)

	_, err = repo.Get(traceID)
	if err == nil {
		t.Error("Expected error when decoding invalid JSON")
	}
}

func TestOpenObserveTraceRepository_Get_NoHits(t *testing.T) {
	traceID, _ := trigger.NewTraceId("1234567890abcdef1234567890abcdef")

	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		return []map[string]any{}, http.StatusOK
	})

	repo := openobserve.NewOpenObserveTraceRepository(
		&openobserve.OpenObserveConfig{
			BaseURL:    mockServer.URL,
			OrgName:    "org-test",
			StreamName: "stream-test",
			Username:   "user",
			Password:   "pass",
		},
		context.Background(),
	)

	_, err := repo.Get(traceID)
	if err == nil {
		t.Error("Expected error when no hits are found for the trace")
	} else if !strings.Contains(err.Error(), "not found in OpenObserve") {
		t.Errorf("Expected 'not found in OpenObserve' error, got: %v", err)
	}
}

/*
type mockIdGenerator struct {
	TraceID string
	SpanID  string
}

func (m *mockIdGenerator) Generate(length int) (string, error) {
	if length == 32 {
		return m.TraceID, nil
	}
	return m.SpanID, nil
}
*/
