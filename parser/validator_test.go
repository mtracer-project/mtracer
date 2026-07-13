package parser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/parser"
)

func ptr[T any](v T) *T {
	return &v
}

func TestExpectedTraceDTO_Validate(t *testing.T) {
	t.Run("defaults checker and ordered", func(t *testing.T) {
		dto := &parser.ExpectedTraceDTO{}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dto.Checker == nil || *dto.Checker != parser.DEFAULT_CHECKER {
			t.Errorf("expected default checker to be 'contains', got %v", dto.Checker)
		}
		if dto.Ordered == nil || *dto.Ordered != parser.DEFAULT_ORDERED {
			t.Errorf("expected default ordered to be true, got %v", dto.Ordered)
		}
	})

	t.Run("invalid max duration", func(t *testing.T) {
		dto := &parser.ExpectedTracePropertiesDTO{
			MaxDuration: ptr(domain.Duration(-1 * time.Second)),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "max duration has to be greater than 0") {
			t.Errorf("expected error about max duration, got: %v", err)
		}
	})

	t.Run("invalid min duration", func(t *testing.T) {
		dto := &parser.ExpectedTracePropertiesDTO{
			MinDuration: ptr(domain.Duration(0)),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "min duration has to be greater than 0") {
			t.Errorf("expected error about min duration, got: %v", err)
		}
	})

	t.Run("invalid span count", func(t *testing.T) {
		dto := &parser.ExpectedTracePropertiesDTO{
			SpanCount: ptr(-1),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "span count has to be greater than or equal to 0") {
			t.Errorf("expected error about span count, got: %v", err)
		}
	})

	t.Run("invalid error count", func(t *testing.T) {
		dto := &parser.ExpectedTracePropertiesDTO{
			ErrorCount: ptr(-2),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "error count has to be greater than or equal to 0") {
			t.Errorf("expected error about error count, got: %v", err)
		}
	})

	t.Run("invalid child span", func(t *testing.T) {
		dto := &parser.ExpectedTraceDTO{
			Spans: []*parser.ExpectedSpanDTO{
				{
					SpanDTO: parser.SpanDTO{
						SpanKind: ptr("invalid-kind"),
					},
				},
			},
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "invalid expected span at index 0") {
			t.Errorf("expected error about child span, got: %v", err)
		}
	})

	t.Run("nil child span", func(t *testing.T) {
		dto := &parser.ExpectedTraceDTO{
			Spans: []*parser.ExpectedSpanDTO{nil},
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "expected span at index 0 cannot be nil") {
			t.Errorf("expected error about nil child span, got: %v", err)
		}
	})
}

func TestExpectedSpanDTO_Validate(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		dto := &parser.ExpectedSpanDTO{
			SpanDTO: parser.SpanDTO{
				SpanKind:   ptr("SERVER"),
				SpanStatus: ptr("ok"),
			},
			MaxDuration: ptr(domain.Duration(5 * time.Second)),
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid kind", func(t *testing.T) {
		dto := &parser.ExpectedSpanDTO{
			SpanDTO: parser.SpanDTO{
				SpanKind: ptr("invalid-span-kind"),
			},
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "invalid span kind") {
			t.Errorf("expected error about kind, got: %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		dto := &parser.ExpectedSpanDTO{
			SpanDTO: parser.SpanDTO{
				SpanStatus: ptr("invalid-status"),
			},
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "invalid span status") {
			t.Errorf("expected error about status, got: %v", err)
		}
	})

	t.Run("invalid max duration", func(t *testing.T) {
		dto := &parser.ExpectedSpanDTO{
			MaxDuration: ptr(domain.Duration(0)),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "max duration has to be greater than 0") {
			t.Errorf("expected error about max duration, got: %v", err)
		}
	})

	t.Run("invalid min duration", func(t *testing.T) {
		dto := &parser.ExpectedSpanDTO{
			MinDuration: ptr(domain.Duration(-10 * time.Millisecond)),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "min duration has to be greater than 0") {
			t.Errorf("expected error about min duration, got: %v", err)
		}
	})

	t.Run("valid kind does not skip status and duration checks", func(t *testing.T) {
		dto := &parser.ExpectedSpanDTO{
			SpanDTO: parser.SpanDTO{
				SpanKind:   ptr("internal"),
				SpanStatus: ptr("invalid-status"),
			},
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "invalid span status") {
			t.Errorf("expected status error to be caught, got: %v", err)
		}

		dto2 := &parser.ExpectedSpanDTO{
			SpanDTO: parser.SpanDTO{
				SpanKind: ptr("internal"),
			},
			MaxDuration: ptr(domain.Duration(0)),
		}
		err2 := dto2.Validate()
		if err2 == nil || !strings.Contains(err2.Error(), "max duration has to be greater than 0") {
			t.Errorf("expected duration error to be caught, got: %v", err2)
		}
	})
}

func TestSpanDTO_Validate(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		dto := &parser.SpanDTO{
			SpanKind:   ptr("client"),
			SpanStatus: ptr("error"),
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid kind", func(t *testing.T) {
		dto := &parser.SpanDTO{
			SpanKind: ptr("db"),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "invalid span kind") {
			t.Errorf("expected error about kind, got: %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		dto := &parser.SpanDTO{
			SpanStatus: ptr("failed"),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "invalid span status") {
			t.Errorf("expected error about status, got: %v", err)
		}
	})

	t.Run("valid kind does not skip status check", func(t *testing.T) {
		dto := &parser.SpanDTO{
			SpanKind:   ptr("client"),
			SpanStatus: ptr("invalid-status"),
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "invalid span status") {
			t.Errorf("expected error about status, got: %v", err)
		}
	})
}

func TestAssertionDTO_Validate(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		dto := &parser.AssertionDTO{
			Name: "check 1",
			Type: "cel",
			Queries: map[string]any{
				"test": "value",
			},
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		dto := &parser.AssertionDTO{
			Queries: map[string]any{"test": "val"},
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "assertion name is required") {
			t.Errorf("expected error about name, got: %v", err)
		}
	})

	t.Run("empty queries", func(t *testing.T) {
		dto := &parser.AssertionDTO{
			Name: "assertion name",
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "assertion queries are required") {
			t.Errorf("expected error about queries, got: %v", err)
		}
	})

	t.Run("default type", func(t *testing.T) {
		dto := &parser.AssertionDTO{
			Name:    "assertion name",
			Queries: map[string]any{"test": "val"},
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dto.Type != parser.DEFAULT_ASSERTION_TYPE {
			t.Errorf("expected default type %q, got %q", parser.DEFAULT_ASSERTION_TYPE, dto.Type)
		}
	})
}

func TestSetupCommandDTO_Validate(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Type: "shell",
			Cmd:  "echo 1",
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty cmd", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Type: "shell",
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "setup command cmd is required") {
			t.Errorf("expected error about cmd, got: %v", err)
		}
	})

	t.Run("default type", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "echo 1",
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dto.Type != parser.DEFAULT_SETUP_COMMAND_TYPE {
			t.Errorf("expected default type %q, got %q", parser.DEFAULT_SETUP_COMMAND_TYPE, dto.Type)
		}
	})
}

func TestTriggerDTO_Validate(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		dto := &parser.TriggerDTO{
			Type: "http",
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil trigger", func(t *testing.T) {
		var dto *parser.TriggerDTO
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "trigger is required") {
			t.Errorf("expected error for nil trigger, got: %v", err)
		}
	})

	t.Run("default type", func(t *testing.T) {
		dto := &parser.TriggerDTO{}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dto.Type != parser.DEFAULT_TRIGGER_TYPE {
			t.Errorf("expected default type %q, got %q", parser.DEFAULT_TRIGGER_TYPE, dto.Type)
		}
	})
}

func TestTestDTO_Validate(t *testing.T) {
	tests := []struct {
		name        string
		dto         *parser.TestDTO
		expectedErr string
		extraChecks func(t *testing.T, dto *parser.TestDTO)
	}{
		{
			name: "missing name",
			dto: &parser.TestDTO{
				Trigger: &parser.TriggerDTO{},
			},
			expectedErr: "test name is required",
		},
		{
			name: "missing trigger",
			dto: &parser.TestDTO{
				Name: "Test 1",
			},
			expectedErr: "trigger is required",
		},
		{
			name: "defaults injection",
			dto: &parser.TestDTO{
				Name:    "Test 1",
				Trigger: &parser.TriggerDTO{},
			},
			expectedErr: "",
			extraChecks: func(t *testing.T, dto *parser.TestDTO) {
				if dto.Timeout == nil || time.Duration(*dto.Timeout) != parser.DEFAULT_TIMEOUT {
					t.Errorf("expected timeout to default to 60s, got %v", dto.Timeout)
				}
				if dto.RetryDelay == nil || time.Duration(*dto.RetryDelay) != parser.DEFAULT_RETRYDELAY {
					t.Errorf("expected retryDelay to default to 1s, got %v", dto.RetryDelay)
				}
				if dto.WaitBeforeFetch == nil || time.Duration(*dto.WaitBeforeFetch) != parser.DEFAULT_WAIT_BEFORE_FETCH {
					t.Errorf("expected waitBeforeFetch to default to 5s, got %v", dto.WaitBeforeFetch)
				}
			},
		},
		{
			name: "invalid timeout",
			dto: &parser.TestDTO{
				Name:    "Test 1",
				Trigger: &parser.TriggerDTO{},
				Timeout: ptr(domain.Duration(0)),
			},
			expectedErr: "timeout has to be greater than 0",
		},
		{
			name: "invalid retry delay",
			dto: &parser.TestDTO{
				Name:       "Test 1",
				Trigger:    &parser.TriggerDTO{},
				RetryDelay: ptr(domain.Duration(-1 * time.Second)),
			},
			expectedErr: "retry delay has to be greater than 0",
		},
		{
			name: "invalid last span",
			dto: &parser.TestDTO{
				Name:    "Test 1",
				Trigger: &parser.TriggerDTO{},
				LastSpan: &parser.ExpectedSpanDTO{
					SpanDTO: parser.SpanDTO{
						SpanKind: ptr("invalid-kind"),
					},
				},
			},
			expectedErr: "invalid expected last span",
		},
		{
			name: "invalid expected trace in list",
			dto: &parser.TestDTO{
				Name:    "Test 1",
				Trigger: &parser.TriggerDTO{},
				ExpectedTraces: []*parser.ExpectedTraceDTO{
					{
						Spans: []*parser.ExpectedSpanDTO{nil},
					},
				},
			},
			expectedErr: "invalid expected trace 1",
		},
		{
			name: "nil setup command in list",
			dto: &parser.TestDTO{
				Name:          "Test 1",
				Trigger:       &parser.TriggerDTO{},
				SetupCommands: []*parser.SetupCommandDTO{nil},
			},
			expectedErr: "setup command 1 cannot be nil",
		},
		{
			name: "invalid setup command in list",
			dto: &parser.TestDTO{
				Name:          "Test 1",
				Trigger:       &parser.TriggerDTO{},
				SetupCommands: []*parser.SetupCommandDTO{{Cmd: ""}},
			},
			expectedErr: "invalid setup command 1",
		},
		{
			name: "nil expected trace in list",
			dto: &parser.TestDTO{
				Name:           "Test 1",
				Trigger:        &parser.TriggerDTO{},
				ExpectedTraces: []*parser.ExpectedTraceDTO{nil},
			},
			expectedErr: "expected trace 1 cannot be nil",
		},
		{
			name: "nil assertion in list",
			dto: &parser.TestDTO{
				Name:       "Test 1",
				Trigger:    &parser.TriggerDTO{},
				Assertions: []*parser.AssertionDTO{nil},
			},
			expectedErr: "assertion 1 cannot be nil",
		},
		{
			name: "invalid assertion in list",
			dto: &parser.TestDTO{
				Name:       "Test 1",
				Trigger:    &parser.TriggerDTO{},
				Assertions: []*parser.AssertionDTO{{Name: ""}},
			},
			expectedErr: "invalid assertion 1",
		},
		{
			name: "invalid expected trace properties",
			dto: &parser.TestDTO{
				Name:    "Test 1",
				Trigger: &parser.TriggerDTO{},
				ExpectedProperties: &parser.ExpectedTracePropertiesDTO{
					SpanCount: ptr(-1),
				},
			},
			expectedErr: "invalid expected trace properties",
		},
		{
			name: "nil post exec check in list",
			dto: &parser.TestDTO{
				Name:           "Test 1",
				Trigger:        &parser.TriggerDTO{},
				PostExecChecks: []*parser.PostExecCheckDTO{nil},
			},
			expectedErr: "post exec check 1 cannot be nil",
		},
		{
			name: "invalid post exec check in list",
			dto: &parser.TestDTO{
				Name:           "Test 1",
				Trigger:        &parser.TriggerDTO{},
				PostExecChecks: []*parser.PostExecCheckDTO{{Name: ""}},
			},
			expectedErr: "invalid post exec check 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.dto.Validate()

			if tc.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedErr) {
					t.Errorf("expected error containing %q, got: %v", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if tc.extraChecks != nil {
				tc.extraChecks(t, tc.dto)
			}
		})
	}
}

func TestPostExecCheckDTO_Validate(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check 1",
			Type: "shell",
			Args: map[string]any{
				"cmd": "echo hello",
			},
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Type: "shell",
			Args: map[string]any{"cmd": "echo hello"},
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "post exec check name is required") {
			t.Errorf("expected error about name, got: %v", err)
		}
	})

	t.Run("empty args", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check 1",
			Type: "shell",
		}
		err := dto.Validate()
		if err == nil || !strings.Contains(err.Error(), "post exec check args are required") {
			t.Errorf("expected error about args, got: %v", err)
		}
	})

	t.Run("default type", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check 1",
			Args: map[string]any{"cmd": "echo hello"},
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dto.Type != parser.DEFAULT_POST_EXEC_CHECK_TYPE {
			t.Errorf("expected default type %q, got %q", parser.DEFAULT_POST_EXEC_CHECK_TYPE, dto.Type)
		}
	})

	t.Run("explicit type is preserved", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check 1",
			Type: "sql",
			Args: map[string]any{"query": "SELECT 1"},
		}
		err := dto.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dto.Type != "sql" {
			t.Errorf("expected type 'sql', got %q", dto.Type)
		}
	})
}
