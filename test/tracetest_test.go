package test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/assertion"
	"github.com/mtracer-project/mtracer/domain"
	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"
	postexecchecks "github.com/mtracer-project/mtracer/postExecChecks"
	setupcommand "github.com/mtracer-project/mtracer/setupCommand"
	"github.com/mtracer-project/mtracer/span"
	testutils "github.com/mtracer-project/mtracer/testUtils"
	"github.com/mtracer-project/mtracer/trace"
	"github.com/mtracer-project/mtracer/trigger"
)

// Mock SetupCommand
type mockSetupCommand struct {
	executeErr   error
	cleanupErr   error
	executeCalls int
	cleanupCalls int
}

func (m *mockSetupCommand) Execute() error {
	m.executeCalls++
	return m.executeErr
}

func (m *mockSetupCommand) Cleanup() error {
	m.cleanupCalls++
	return m.cleanupErr
}

// Mock Trigger
type mockTrigger struct {
	traceId      trigger.TraceId
	err          error
	triggerCalls int
}

func (m *mockTrigger) Trigger() (trigger.TraceId, error) {
	m.triggerCalls++
	return m.traceId, m.err
}

func (m *mockTrigger) Example() string {
	return "example"
}

func (m *mockTrigger) Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error {
	return nil
}

type mockIdGenerator struct{}

func (m *mockIdGenerator) Generate(length int) (string, error) {
	return "0123456789abcdef0123456789abcdef", nil
}

// Mock TraceAdapter
type mockTraceAdapter struct {
	trace           *trace.Trace
	err             error
	fetchCalls      int
	capturedTraceId trigger.TraceId
}

func (m *mockTraceAdapter) Fetch(traceId trigger.TraceId, timeout time.Duration, retryDelay time.Duration, lastSpan *span.ExpectedSpan) (*trace.Trace, error) {
	m.fetchCalls++
	m.capturedTraceId = traceId
	return m.trace, m.err
}

// Mock Assertion
type mockAssertion struct {
	passed      bool
	err         error
	assertCalls int
}

func (m *mockAssertion) Assert(t *trace.Trace) (bool, error) {
	m.assertCalls++
	return m.passed, m.err
}

// Mock PostExecCheck
type mockPostExecCheck struct {
	passed     bool
	err        error
	checkCalls int
}

func (m *mockPostExecCheck) Check() (bool, error) {
	m.checkCalls++
	return m.passed, m.err
}

func TestNewTraceTest(t *testing.T) {
	timeout := domain.Duration(10 * time.Second)
	retryDelay := domain.Duration(1 * time.Second)
	waitBeforeFetch := domain.Duration(500 * time.Millisecond)

	validDTO := &parser.TestDTO{
		Name:            "Test 1",
		Description:     "Test Description",
		Timeout:         &timeout,
		RetryDelay:      &retryDelay,
		WaitBeforeFetch: &waitBeforeFetch,
		SetupCommands: []*parser.SetupCommandDTO{
			{
				Type: "shell",
				Cmd:  "echo hello",
				CleanupCmd: &parser.CleanupCommandDTO{
					Cmd: "echo cleanup",
				},
			},
		},
		Trigger: &parser.TriggerDTO{
			Type: "http",
			Args: map[string]any{
				"url": "http://localhost:8080/test",
			},
		},
		Assertions: []*parser.AssertionDTO{
			{
				Type: "cel",
				Queries: map[string]any{
					"q1": "trace.spanCount > 0",
				},
			},
		},
	}

	// 1. Success case
	idGen := &mockIdGenerator{}
	adapter := &mockTraceAdapter{}
	tt, err := NewTraceTest(validDTO, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if tt.name != "Test 1" {
		t.Errorf("Expected name 'Test 1', got %q", tt.name)
	}

	// 2. Missing timeout
	dtoNoTimeout := *validDTO
	dtoNoTimeout.Timeout = nil
	_, err = NewTraceTest(&dtoNoTimeout, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err == nil || err.Error() != "timeout is required" {
		t.Errorf("Expected 'timeout is required' error, got %v", err)
	}

	// 3. Missing retry delay
	dtoNoRetry := *validDTO
	dtoNoRetry.RetryDelay = nil
	_, err = NewTraceTest(&dtoNoRetry, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err == nil || err.Error() != "retry delay is required" {
		t.Errorf("Expected 'retry delay is required' error, got %v", err)
	}

	// 4. Missing wait before fetch
	dtoNoWait := *validDTO
	dtoNoWait.WaitBeforeFetch = nil
	_, err = NewTraceTest(&dtoNoWait, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err == nil || err.Error() != "wait before fetch is required" {
		t.Errorf("Expected 'wait before fetch is required' error, got %v", err)
	}

	// 5. Setup command construction error
	dtoBadSetup := *validDTO
	dtoBadSetup.SetupCommands = []*parser.SetupCommandDTO{
		{
			Type: "invalid-type",
		},
	}
	_, err = NewTraceTest(&dtoBadSetup, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err == nil {
		t.Error("Expected setup commands error, got nil")
	}

	// 6. Trigger construction error
	dtoBadTrigger := *validDTO
	dtoBadTrigger.Trigger = &parser.TriggerDTO{
		Type: "invalid-type",
	}
	_, err = NewTraceTest(&dtoBadTrigger, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err == nil {
		t.Error("Expected trigger construction error, got nil")
	}

	// 7. Assertion construction error
	dtoBadAssertion := *validDTO
	dtoBadAssertion.Assertions = []*parser.AssertionDTO{
		{
			Type: "invalid-type",
		},
	}
	_, err = NewTraceTest(&dtoBadAssertion, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err == nil {
		t.Error("Expected assertion construction error, got nil")
	}

	// 8. PostExecCheck construction error (unsupported type)
	dtoBadPostExec := *validDTO
	dtoBadPostExec.PostExecChecks = []*parser.PostExecCheckDTO{
		{
			Name: "bad-check",
			Type: "invalid-type",
			Args: map[string]any{},
		},
	}
	_, err = NewTraceTest(&dtoBadPostExec, idGen, nil, adapter, TraceTestOptions{}, "", context.Background())
	if err == nil {
		t.Error("Expected post exec check construction error, got nil")
	}
}

func TestTraceTest_Run_AllSuccess(t *testing.T) {
	cmd1 := &mockSetupCommand{}
	cmd2 := &mockSetupCommand{}
	trig := &mockTrigger{traceId: trigger.TraceId("12345678901234567890123456789012")}
	adapter := &mockTraceAdapter{
		trace: &trace.Trace{
			TraceId: "12345678901234567890123456789012",
		},
	}
	assert1 := &mockAssertion{passed: true}
	assert2 := &mockAssertion{passed: true}

	tt := &TraceTest{
		name:          "Run Success Test",
		setupCommands: []setupcommand.SetupCommand{cmd1, cmd2},
		trigger:       trig,
		adapter:       adapter,
		assertions:    []assertion.Assertion{assert1, assert2},
	}

	res := tt.Run()
	if !res.Passed {
		t.Fatalf("Expected run to succeed, got failure: %+v", res.Args)
	}

	if cmd1.executeCalls != 1 || cmd1.cleanupCalls != 1 {
		t.Errorf("Expected cmd1 1 execute, 1 cleanup; got %d, %d", cmd1.executeCalls, cmd1.cleanupCalls)
	}
	if cmd2.executeCalls != 1 || cmd2.cleanupCalls != 1 {
		t.Errorf("Expected cmd2 1 execute, 1 cleanup; got %d, %d", cmd2.executeCalls, cmd2.cleanupCalls)
	}
	if trig.triggerCalls != 1 {
		t.Errorf("Expected 1 trigger call, got %d", trig.triggerCalls)
	}
	if adapter.fetchCalls != 1 || adapter.capturedTraceId != "12345678901234567890123456789012" {
		t.Errorf("Expected 1 fetch call with trace ID '12345678901234567890123456789012', got %d calls, trace ID %q", adapter.fetchCalls, adapter.capturedTraceId)
	}
	if assert1.assertCalls != 1 {
		t.Errorf("Expected 1 assert1 call, got %d", assert1.assertCalls)
	}
	if assert2.assertCalls != 1 {
		t.Errorf("Expected 1 assert2 call, got %d", assert2.assertCalls)
	}

	if tt.Name() != "Run Success Test" {
		t.Errorf("Expected Name() to be 'Run Success Test', got %q", tt.Name())
	}
}

func TestTraceTest_Run_SetupFailure(t *testing.T) {
	cmd1 := &mockSetupCommand{}
	cmd2 := &mockSetupCommand{executeErr: errors.New("setup failed")}
	cmd3 := &mockSetupCommand{}
	trig := &mockTrigger{}

	tt := &TraceTest{
		setupCommands: []setupcommand.SetupCommand{cmd1, cmd2, cmd3},
		trigger:       trig,
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail due to setup command failure")
	}

	if cmd1.executeCalls != 1 || cmd1.cleanupCalls != 1 {
		t.Errorf("Expected cmd1 1 execute, 1 cleanup; got %d, %d", cmd1.executeCalls, cmd1.cleanupCalls)
	}
	// cmd2 failed during execution, but its Cleanup should still be called via defer
	if cmd2.executeCalls != 1 || cmd2.cleanupCalls != 1 {
		t.Errorf("Expected cmd2 1 execute, 1 cleanup; got %d, %d", cmd2.executeCalls, cmd2.cleanupCalls)
	}
	// cmd3 was never executed, so its execute should be 0, cleanup should be 0
	if cmd3.executeCalls != 0 || cmd3.cleanupCalls != 0 {
		t.Errorf("Expected cmd3 0 execute, 0 cleanup; got %d, %d", cmd3.executeCalls, cmd3.cleanupCalls)
	}
	if trig.triggerCalls != 0 {
		t.Errorf("Expected trigger to not be called on setup failure, got %d calls", trig.triggerCalls)
	}
}

func TestTraceTest_Run_TriggerFailure(t *testing.T) {
	cmd := &mockSetupCommand{}
	trig := &mockTrigger{err: errors.New("trigger failed")}
	adapter := &mockTraceAdapter{}

	tt := &TraceTest{
		setupCommands: []setupcommand.SetupCommand{cmd},
		trigger:       trig,
		adapter:       adapter,
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail due to trigger failure")
	}

	if cmd.executeCalls != 1 || cmd.cleanupCalls != 1 {
		t.Errorf("Expected setup command executed and cleaned up, got %d, %d", cmd.executeCalls, cmd.cleanupCalls)
	}
	if trig.triggerCalls != 1 {
		t.Errorf("Expected 1 trigger call, got %d", trig.triggerCalls)
	}
	if adapter.fetchCalls != 0 {
		t.Errorf("Expected no fetch calls on trigger failure, got %d", adapter.fetchCalls)
	}
}

func TestTraceTest_Run_FetchFailure(t *testing.T) {
	trig := &mockTrigger{traceId: trigger.TraceId("12345678901234567890123456789012")}
	adapter := &mockTraceAdapter{err: errors.New("fetch failed")}
	assert1 := &mockAssertion{}

	tt := &TraceTest{
		trigger:    trig,
		adapter:    adapter,
		assertions: []assertion.Assertion{assert1},
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail due to fetch failure")
	}

	if adapter.fetchCalls != 1 {
		t.Errorf("Expected 1 fetch call, got %d", adapter.fetchCalls)
	}
	if assert1.assertCalls != 0 {
		t.Errorf("Expected no assertions run on fetch failure, got %d", assert1.assertCalls)
	}
}

func TestTraceTest_Run_ComparisonFailure(t *testing.T) {
	trig := &mockTrigger{traceId: trigger.TraceId("12345678901234567890123456789012")}
	actualTrace := &trace.Trace{
		TraceId:    "12345678901234567890123456789012",
		SpanCount:  1,
		ErrorCount: 0,
	}
	adapter := &mockTraceAdapter{trace: actualTrace}

	// Create expected properties that won't match (e.g. requires errorCount = 1)
	errCount := 1
	expProperties := trace.NewExpectedTraceProperties(&parser.ExpectedTracePropertiesDTO{
		ErrorCount: &errCount,
	})

	assert1 := &mockAssertion{}

	tt := &TraceTest{
		trigger:            trig,
		adapter:            adapter,
		expectedProperties: expProperties,
		assertions:         []assertion.Assertion{assert1},
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail due to comparison failure")
	}

	if assert1.assertCalls != 0 {
		t.Errorf("Expected no assertions run on comparison failure, got %d", assert1.assertCalls)
	}
}

func TestTraceTest_Run_AssertionError(t *testing.T) {
	trig := &mockTrigger{traceId: trigger.TraceId("12345678901234567890123456789012")}
	adapter := &mockTraceAdapter{trace: &trace.Trace{}}
	assert1 := &mockAssertion{err: errors.New("assertion runtime error")}

	tt := &TraceTest{
		trigger:    trig,
		adapter:    adapter,
		assertions: []assertion.Assertion{assert1},
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail due to assertion error")
	}

	if assert1.assertCalls != 1 {
		t.Errorf("Expected 1 assertion call, got %d", assert1.assertCalls)
	}
}

func TestTraceTest_Run_AssertionFailed(t *testing.T) {
	trig := &mockTrigger{traceId: trigger.TraceId("12345678901234567890123456789012")}
	adapter := &mockTraceAdapter{trace: &trace.Trace{}}
	assert1 := &mockAssertion{passed: false}

	tt := &TraceTest{
		trigger:    trig,
		adapter:    adapter,
		assertions: []assertion.Assertion{assert1},
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail because assertion returned false")
	}

	if assert1.assertCalls != 1 {
		t.Errorf("Expected 1 assertion call, got %d", assert1.assertCalls)
	}
}

func TestTraceTest_Run_PostExecCheckError(t *testing.T) {
	trig := &mockTrigger{traceId: trigger.TraceId("12345678901234567890123456789012")}
	adapter := &mockTraceAdapter{trace: &trace.Trace{}}
	check1 := &mockPostExecCheck{err: errors.New("check runtime error")}

	tt := &TraceTest{
		trigger:        trig,
		adapter:        adapter,
		postExecChecks: []postexecchecks.PostExecCheck{check1},
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail due to post exec check error")
	}

	if check1.checkCalls != 1 {
		t.Errorf("Expected 1 check call, got %d", check1.checkCalls)
	}
}

func TestTraceTest_Run_PostExecCheckFailed(t *testing.T) {
	trig := &mockTrigger{traceId: trigger.TraceId("12345678901234567890123456789012")}
	adapter := &mockTraceAdapter{trace: &trace.Trace{}}
	assert1 := &mockAssertion{passed: true}
	check1 := &mockPostExecCheck{passed: false}

	tt := &TraceTest{
		trigger:        trig,
		adapter:        adapter,
		assertions:     []assertion.Assertion{assert1},
		postExecChecks: []postexecchecks.PostExecCheck{check1},
	}

	res := tt.Run()
	if res.Passed {
		t.Fatal("Expected run to fail because post exec check returned false")
	}

	if assert1.assertCalls != 1 {
		t.Errorf("Expected assertions to run before post exec checks, got %d assertion calls", assert1.assertCalls)
	}
	if check1.checkCalls != 1 {
		t.Errorf("Expected 1 check call, got %d", check1.checkCalls)
	}
}

func TestTraceTest_Run_DockerSetup(t *testing.T) {
	timeout := domain.Duration(10 * time.Second)
	retryDelay := domain.Duration(1 * time.Second)
	waitBeforeFetch := domain.Duration(500 * time.Millisecond)

	dto := &parser.TestDTO{
		Name:            "TraceTest with Docker",
		Description:     "Description",
		Timeout:         &timeout,
		RetryDelay:      &retryDelay,
		WaitBeforeFetch: &waitBeforeFetch,
		SetupCommands: []*parser.SetupCommandDTO{
			{
				Type: "docker",
				Cmd:  "killcontainer",
				Args: map[string]any{
					"containerId": "test-container-docker-run",
				},
			},
		},
		Trigger: &parser.TriggerDTO{
			Type: "traceid",
			Args: map[string]any{
				"traceId": "12345678901234567890123456789012",
			},
		},
	}

	idGen := &mockIdGenerator{}
	adapter := &mockTraceAdapter{
		trace: &trace.Trace{
			TraceId: "12345678901234567890123456789012",
		},
	}

	var calledKill, calledStart bool
	cli := testutils.NewMockDockerClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container-docker-run/kill") {
			calledKill = true
		}
		if req.Method == "POST" && strings.Contains(req.URL.Path, "/containers/test-container-docker-run/start") {
			calledStart = true
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		}, nil
	})

	tt, err := NewTraceTest(dto, idGen, cli, adapter, TraceTestOptions{}, "", context.Background())
	if err != nil {
		t.Fatalf("unexpected error creating trace test: %v", err)
	}

	res := tt.Run()
	if !res.Passed {
		t.Fatalf("expected run to succeed, got: %+v", res.Args)
	}

	if !calledKill {
		t.Error("expected docker kill container API to be called during Execute")
	}
	if !calledStart {
		t.Error("expected docker start container API to be called during Cleanup")
	}
}
