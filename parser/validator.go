package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/mtrace-project/mtrace/domain"
)

const (
	DEFAULT_CHECKER = "contains"
	DEFAULT_ORDERED = true

	DEFAULT_TIMEOUT           = time.Duration(60) * time.Second
	DEFAULT_RETRYDELAY        = time.Duration(1) * time.Second
	DEFAULT_WAIT_BEFORE_FETCH = time.Duration(5) * time.Second

	DEFAULT_ASSERTION_TYPE = "cel"

	DEFAULT_SETUP_COMMAND_TYPE   = "shell"
	DEFAULT_POST_EXEC_CHECK_TYPE = "shell"

	DEFAULT_TRIGGER_TYPE = "http"
)

func (t *ExpectedTraceDTO) Validate() error {
	if t.Checker == nil {
		checker := DEFAULT_CHECKER
		t.Checker = &checker
	}
	if t.Ordered == nil {
		ordered := DEFAULT_ORDERED
		t.Ordered = &ordered
	}

	for i, spanDTO := range t.Spans {
		if spanDTO == nil {
			return fmt.Errorf("expected span at index %d cannot be nil", i)
		}
		if err := spanDTO.Validate(); err != nil {
			return fmt.Errorf("invalid expected span at index %d: %w", i, err)
		}
	}

	return nil
}

func (p *ExpectedTracePropertiesDTO) Validate() error {
	if p.MaxDuration != nil && *p.MaxDuration <= 0 {
		return fmt.Errorf("max duration has to be greater than 0")
	}

	if p.MinDuration != nil && *p.MinDuration <= 0 {
		return fmt.Errorf("min duration has to be greater than 0")
	}

	if p.SpanCount != nil && *p.SpanCount < 0 {
		return fmt.Errorf("span count has to be greater than or equal to 0")
	}

	if p.ErrorCount != nil && *p.ErrorCount < 0 {
		return fmt.Errorf("error count has to be greater than or equal to 0")
	}

	return nil
}

func (s *ExpectedSpanDTO) Validate() error {
	if s.SpanKind != nil {
		switch strings.ToLower(*s.SpanKind) {
		case "internal", "server", "client", "producer", "consumer", "unset", "unspecified":
			// valid
		default:
			return fmt.Errorf("invalid span kind: %s", *s.SpanKind)
		}
	}

	if s.SpanStatus != nil {
		switch strings.ToLower(*s.SpanStatus) {
		case "ok", "error", "unset":
			// valid
		default:
			return fmt.Errorf("invalid span status: %s", *s.SpanStatus)
		}
	}

	if s.MaxDuration != nil && *s.MaxDuration <= 0 {
		return fmt.Errorf("max duration has to be greater than 0")
	}

	if s.MinDuration != nil && *s.MinDuration <= 0 {
		return fmt.Errorf("min duration has to be greater than 0")
	}

	return nil
}

func (s *SpanDTO) Validate() error {
	if s.SpanKind != nil {
		switch strings.ToLower(*s.SpanKind) {
		case "internal", "server", "client", "producer", "consumer", "unset", "unspecified":
			// valid
		default:
			return fmt.Errorf("invalid span kind: %s", *s.SpanKind)
		}
	}

	if s.SpanStatus != nil {
		switch strings.ToLower(*s.SpanStatus) {
		case "ok", "error", "unset":
			// valid
		default:
			return fmt.Errorf("invalid span status: %s", *s.SpanStatus)
		}
	}

	return nil
}

func (a *AssertionDTO) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("assertion name is required")
	}

	if a.Type == "" {
		a.Type = DEFAULT_ASSERTION_TYPE
	}

	if len(a.Queries) == 0 {
		return fmt.Errorf("assertion queries are required")
	}

	return nil
}

func (p *PostExecCheckDTO) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("post exec check name is required")
	}

	if p.Type == "" {
		p.Type = DEFAULT_POST_EXEC_CHECK_TYPE
	}

	if len(p.Args) == 0 {
		return fmt.Errorf("post exec check args are required")
	}

	return nil
}

func (s *SetupCommandDTO) Validate() error {
	if s.Type == "" {
		s.Type = DEFAULT_SETUP_COMMAND_TYPE
	}

	if s.Cmd == "" {
		return fmt.Errorf("setup command cmd is required")
	}

	return nil
}

func (t *TriggerDTO) Validate() error {
	if t == nil {
		return fmt.Errorf("trigger is required")
	}

	if t.Type == "" {
		t.Type = DEFAULT_TRIGGER_TYPE
	}
	return nil
}

func (t *TestDTO) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("test name is required")
	}

	for i, setupCmd := range t.SetupCommands {
		if setupCmd == nil {
			return fmt.Errorf("setup command %d cannot be nil", i+1)
		}
		if err := setupCmd.Validate(); err != nil {
			return fmt.Errorf("invalid setup command %d: %w", i+1, err)
		}
	}

	if t.Trigger == nil {
		return fmt.Errorf("trigger is required")
	}

	if err := t.Trigger.Validate(); err != nil {
		return fmt.Errorf("invalid trigger: %w", err)
	}

	if t.WaitBeforeFetch == nil {
		waitBeforeFetch := domain.FromTimeDuration(DEFAULT_WAIT_BEFORE_FETCH)
		t.WaitBeforeFetch = &waitBeforeFetch
	}

	for i, expectedTrace := range t.ExpectedTraces {
		if expectedTrace == nil {
			return fmt.Errorf("expected trace %d cannot be nil", i+1)
		}
		if err := expectedTrace.Validate(); err != nil {
			return fmt.Errorf("invalid expected trace %d: %w", i+1, err)
		}
	}

	if t.ExpectedProperties != nil {
		if err := t.ExpectedProperties.Validate(); err != nil {
			return fmt.Errorf("invalid expected trace properties: %w", err)
		}
	}

	for i, assertion := range t.Assertions {
		if assertion == nil {
			return fmt.Errorf("assertion %d cannot be nil", i+1)
		}
		if err := assertion.Validate(); err != nil {
			return fmt.Errorf("invalid assertion %d: %w", i+1, err)
		}
	}

	for i, postExecCheck := range t.PostExecChecks {
		if postExecCheck == nil {
			return fmt.Errorf("post exec check %d cannot be nil", i+1)
		}
		if err := postExecCheck.Validate(); err != nil {
			return fmt.Errorf("invalid post exec check %d: %w", i+1, err)
		}
	}

	if t.LastSpan != nil {
		if err := t.LastSpan.Validate(); err != nil {
			return fmt.Errorf("invalid expected last span: %w", err)
		}
	}

	if t.Timeout == nil {
		timeout := domain.FromTimeDuration(DEFAULT_TIMEOUT)
		t.Timeout = &timeout
	}

	if *t.Timeout <= 0 {
		return fmt.Errorf("timeout has to be greater than 0")
	}

	if t.RetryDelay == nil {
		retryDelay := domain.FromTimeDuration(DEFAULT_RETRYDELAY)
		t.RetryDelay = &retryDelay
	}

	if *t.RetryDelay <= 0 {
		return fmt.Errorf("retry delay has to be greater than 0")
	}

	return nil
}
