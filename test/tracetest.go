package test

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mtrace-project/mtrace/assertion"
	idgenerator "github.com/mtrace-project/mtrace/idGenerator"
	"github.com/mtrace-project/mtrace/parser"
	postexecchecks "github.com/mtrace-project/mtrace/postExecChecks"
	setupcommand "github.com/mtrace-project/mtrace/setupCommand"
	"github.com/mtrace-project/mtrace/span"
	"github.com/mtrace-project/mtrace/trace"
	"github.com/mtrace-project/mtrace/trigger"

	"github.com/moby/moby/client"
)

type TraceTest struct {
	name               string
	description        string
	setupCommands      []setupcommand.SetupCommand
	trigger            trigger.Trigger
	waitBeforeFetch    time.Duration
	expectedProperties *trace.ExpectedTraceProperties
	expectedTraces     []*trace.ExpectedTrace
	assertions         []assertion.Assertion
	postExecChecks     []postexecchecks.PostExecCheck
	lastSpan           *span.ExpectedSpan
	timeout            time.Duration
	retryDelay         time.Duration

	opts TraceTestOptions

	idGenerator idgenerator.IdGenerator
	adapter     trace.TraceAdapter
}

type TraceTestOptions struct {
	CollectTrace bool
}

func (t *TraceTest) Run() *TestResult {
	start := time.Now()

	// Execute setup commands
	for i, cmd := range t.setupCommands {
		err := cmd.Execute()
		defer func(c setupcommand.SetupCommand) {
			if errClean := c.Cleanup(); errClean != nil {
				slog.Debug("Setup command cleanup failed", "error", errClean)
			}
		}(cmd)
		if err != nil {
			return NewTestResult(
				false,
				time.Since(start),
				nil,
				[]any{"message", fmt.Sprintf("Setup command %d failed", i+1), "error", err},
			)
		}
	}

	// Trigger the test and capture the trace ID
	traceId, err := t.trigger.Trigger()
	if err != nil {
		return NewTestResult(
			false,
			time.Since(start),
			nil,
			[]any{"message", "Trigger execution failed", "error", err},
		)
	}

	// Wait for a given amount of time to ensure the trace is available in the observability backend
	time.Sleep(t.waitBeforeFetch)

	// Fetch the actual trace from the observability backend
	actualTrace, err := t.adapter.Fetch(traceId, t.timeout, t.retryDelay, t.lastSpan)
	if err != nil {
		return NewTestResult(
			false,
			time.Since(start),
			nil,
			[]any{"message", "Failed to fetch the trace", "error", err},
		)
	}

	// Store the actual trace to be returned only if enabled
	var collectedTrace *trace.Trace
	if t.opts.CollectTrace {
		collectedTrace = actualTrace
	}

	// Compare trace properties if expected properties are provided
	equal, reason := actualTrace.CompareProperties(t.expectedProperties)
	if !equal {
		return NewTestResult(
			false,
			time.Since(start),
			collectedTrace,
			[]any{"message", "Trace properties do not match", "traceId", traceId, "reason", reason, "expectedProperties", t.expectedProperties, "actualTrace", actualTrace},
		)
	}

	// Compare the actual trace with the provided expected trace
	for i, expTrace := range t.expectedTraces {
		equal, reason := actualTrace.Compare(expTrace)
		if !equal {
			return NewTestResult(
				false,
				time.Since(start),
				collectedTrace,
				[]any{"message", fmt.Sprintf("Trace %d comparison failed", i+1), "traceId", traceId, "reason", reason, "expectedTrace", expTrace, "actualTrace", actualTrace},
			)
		}
	}

	// Run assertions
	for i, assertion := range t.assertions {
		passed, err := assertion.Assert(actualTrace)
		if err != nil {
			return NewTestResult(
				false,
				time.Since(start),
				collectedTrace,
				[]any{"message", fmt.Sprintf("Assertion %d execution failed", i+1), "traceId", traceId, "error", err},
			)
		}
		if !passed {
			return NewTestResult(
				false,
				time.Since(start),
				collectedTrace,
				[]any{"message", fmt.Sprintf("Assertion %d failed", i+1), "traceId", traceId},
			)
		}
	}

	// Run post-execution checks
	for i, check := range t.postExecChecks {
		passed, err := check.Check()
		if err != nil {
			return NewTestResult(
				false,
				time.Since(start),
				collectedTrace,
				[]any{"message", fmt.Sprintf("Post-execution check %d execution failed", i+1), "traceId", traceId, "error", err},
			)
		}
		if !passed {
			return NewTestResult(
				false,
				time.Since(start),
				collectedTrace,
				[]any{"message", fmt.Sprintf("Post-execution check %d failed", i+1), "traceId", traceId},
			)
		}
	}

	return NewTestResult(
		true,
		time.Since(start),
		collectedTrace,
		[]any{"message", "Test passed successfully", "traceId", traceId},
	)
}

func (t *TraceTest) Name() string {
	return t.name
}

func NewTraceTest(dto *parser.TestDTO, idGenerator idgenerator.IdGenerator, client *client.Client, adapter trace.TraceAdapter, opts TraceTestOptions, baseDir string, ctx context.Context) (*TraceTest, error) {
	setupCommands, err := setupcommand.NewSetupCommands(dto.SetupCommands, client, baseDir, ctx)
	if err != nil {
		return nil, err
	}

	trigger, err := trigger.NewTrigger(dto.Trigger, idGenerator, baseDir, ctx)
	if err != nil {
		return nil, err
	}

	lastSpan := span.NewExpectedSpan(dto.LastSpan)

	assertions, err := assertion.NewAssertions(dto.Assertions)
	if err != nil {
		return nil, err
	}

	postExecChecks, err := postexecchecks.NewPostExecChecks(dto.PostExecChecks, baseDir, ctx)
	if err != nil {
		return nil, err
	}

	if dto.Timeout == nil {
		return nil, fmt.Errorf("timeout is required")
	}

	if dto.RetryDelay == nil {
		return nil, fmt.Errorf("retry delay is required")
	}

	if dto.WaitBeforeFetch == nil {
		return nil, fmt.Errorf("wait before fetch is required")
	}

	test := &TraceTest{
		name:               dto.Name,
		description:        dto.Description,
		setupCommands:      setupCommands,
		trigger:            trigger,
		waitBeforeFetch:    *dto.WaitBeforeFetch.ToTimeDuration(),
		expectedProperties: trace.NewExpectedTraceProperties(dto.ExpectedProperties),
		expectedTraces:     trace.NewExpectedTraces(dto.ExpectedTraces),
		assertions:         assertions,
		postExecChecks:     postExecChecks,
		lastSpan:           lastSpan,
		timeout:            *dto.Timeout.ToTimeDuration(),
		retryDelay:         *dto.RetryDelay.ToTimeDuration(),
		opts:               opts,
		idGenerator:        idGenerator,
		adapter:            adapter,
	}

	return test, nil
}
