package test

import (
	"time"

	"github.com/mtrace-project/mtrace/trace"
)

type Test interface {
	Run() *TestResult
	Name() string
}

type TestSuite struct {
	Name    string
	Results []*TestResult
}

func NewTestSuite(name string, results []*TestResult) *TestSuite {
	return &TestSuite{
		Name:    name,
		Results: results,
	}
}

type TestResult struct {
	Passed   bool
	Duration time.Duration
	Trace    *trace.Trace
	Args     []any // we expect to be used as key-value pairs, e.g., "traceId", "12345", "message", "Test passed"
}

func NewTestResult(passed bool, duration time.Duration, trace *trace.Trace, args []any) *TestResult {
	return &TestResult{
		Passed:   passed,
		Duration: duration,
		Trace:    trace,
		Args:     args,
	}
}
