package assertion_test

import (
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/assertion"
	"github.com/mtrace-project/mtrace/parser"
	"github.com/mtrace-project/mtrace/span"
	"github.com/mtrace-project/mtrace/trace"
	"github.com/mtrace-project/mtrace/trigger"
)

func TestNewAssertion_Factory(t *testing.T) {
	// Case 1: CEL type
	dtoCel := &parser.AssertionDTO{
		Type: "cel",
	}
	ast, err := assertion.NewAssertion(dtoCel)
	if err != nil {
		t.Fatalf("Unexpected error creating CEL assertion: %v", err)
	}
	if ast == nil {
		t.Fatal("Expected non-nil CEL assertion instance")
	}

	// Case 2: Unsupported type
	dtoBad := &parser.AssertionDTO{
		Type: "unsupported",
	}
	_, err = assertion.NewAssertion(dtoBad)
	if err == nil {
		t.Error("Expected error for unsupported assertion type")
	} else if !strings.Contains(err.Error(), "unsupported assertion type") {
		t.Errorf("Expected unsupported error message, got: %v", err)
	}
}

func TestNewAssertions_Batch(t *testing.T) {
	dtos := []*parser.AssertionDTO{
		{Type: "cel"},
		{Type: "cel"},
	}

	assertions, err := assertion.NewAssertions(dtos)
	if err != nil {
		t.Fatalf("Unexpected error in NewAssertions: %v", err)
	}
	if len(assertions) != 2 {
		t.Errorf("Expected 2 assertions, got %d", len(assertions))
	}

	// Case with unsupported type in batch
	dtosBad := []*parser.AssertionDTO{
		{Type: "cel"},
		{Type: "unsupported"},
	}
	_, err = assertion.NewAssertions(dtosBad)
	if err == nil {
		t.Error("Expected batch constructor to return error")
	}
}

func TestCelAssertion_Assert(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")
	tr := &trace.Trace{
		TraceId:    traceID,
		SpanCount:  3,
		ErrorCount: 0,
	}

	// Case 1: nil trace passed to Assert
	dto := &parser.AssertionDTO{
		Name: "nil-check",
		Type: "cel",
		Queries: map[string]any{
			"q1": "trace.spanCount == 3",
		},
	}
	celAst, err := assertion.NewCelAssertion(dto)
	if err != nil {
		t.Fatalf("NewCelAssertion failed: %v", err)
	}
	ok, err := celAst.Assert(nil)
	if ok || err == nil || err.Error() != "trace is nil" {
		t.Errorf("Expected fail with nil trace, got ok=%v, err=%v", ok, err)
	}

	// Case 2: Successful CEL assertion
	dtoMatch := &parser.AssertionDTO{
		Name: "success-check",
		Type: "cel",
		Queries: map[string]any{
			"query1": "trace.spanCount == 3",
			"query2": "trace.errorCount == 0",
		},
	}
	celMatch, err := assertion.NewCelAssertion(dtoMatch)
	if err != nil {
		t.Fatalf("NewCelAssertion failed: %v", err)
	}
	ok, err = celMatch.Assert(tr)
	if !ok || err != nil {
		t.Errorf("Expected assertions to pass, got ok=%v, err=%v", ok, err)
	}

	// Case 3: Failed CEL assertion (falsy expression)
	dtoFail := &parser.AssertionDTO{
		Name: "fail-check",
		Type: "cel",
		Queries: map[string]any{
			"query1": "trace.spanCount == 10",
		},
	}
	celFail, err := assertion.NewCelAssertion(dtoFail)
	if err != nil {
		t.Fatalf("NewCelAssertion failed: %v", err)
	}
	ok, err = celFail.Assert(tr)
	if ok || err == nil || !strings.Contains(err.Error(), "assertion failed: fail-check") {
		t.Errorf("Expected assertion failure, got ok=%v, err=%v", ok, err)
	}

	// Case 4: Non-boolean output program expression (should compile but fail evaluation)
	dtoIntResult := &parser.AssertionDTO{
		Name: "int-result-check",
		Type: "cel",
		Queries: map[string]any{
			"query1": "trace.spanCount", // Returns int, not bool
		},
	}
	_, err = assertion.NewCelAssertion(dtoIntResult)
	if err == nil {
		t.Error("Expected compilation error for non-boolean query output type")
	}

	// Case 5: Syntax / Compile error
	dtoSyntaxError := &parser.AssertionDTO{
		Name: "syntax-error-check",
		Type: "cel",
		Queries: map[string]any{
			"query1": "trace.spanCount ==", // Malformed syntax
		},
	}
	_, err = assertion.NewCelAssertion(dtoSyntaxError)
	if err == nil {
		t.Error("Expected syntax compilation error")
	}
}

func TestCelAssertion_Attributes(t *testing.T) {
	traceID, _ := trigger.NewTraceId("11112222333344445555666677778888")
	tr := &trace.Trace{
		TraceId:    traceID,
		SpanCount:  1,
		ErrorCount: 0,
		Spans: []*span.Span{
			{
				SpanId: "s1",
				Attributes: map[string]any{
					"http.method":      "GET",
					"http.status_code": 200,
					"consumer":         true,
				},
			},
		},
	}

	dtoMatch := &parser.AssertionDTO{
		Name: "attributes-check",
		Type: "cel",
		Queries: map[string]any{
			"query1": "trace.spans[0].attributes['http.method'] == 'GET'",
			"query2": "trace.spans[0].attributes['http.status_code'] == 200",
			"query3": "trace.spans[0].attributes['consumer'] == true",
		},
	}
	celMatch, err := assertion.NewCelAssertion(dtoMatch)
	if err != nil {
		t.Fatalf("NewCelAssertion failed: %v", err)
	}
	ok, err := celMatch.Assert(tr)
	if !ok || err != nil {
		t.Errorf("Expected assertions to pass, got ok=%v, err=%v", ok, err)
	}

	// Case 2: Attribute mismatch
	dtoMismatch := &parser.AssertionDTO{
		Name: "attributes-mismatch",
		Type: "cel",
		Queries: map[string]any{
			"query1": "trace.spans[0].attributes['http.method'] == 'POST'",
		},
	}
	celMismatch, err := assertion.NewCelAssertion(dtoMismatch)
	if err != nil {
		t.Fatalf("NewCelAssertion failed: %v", err)
	}
	ok, err = celMismatch.Assert(tr)
	if ok || err == nil {
		t.Errorf("Expected assertion to fail for attribute mismatch, got ok=%v, err=%v", ok, err)
	}

	// Case 3: Span without attributes (nil map)
	trNoAttrs := &trace.Trace{
		TraceId:    traceID,
		SpanCount:  1,
		ErrorCount: 0,
		Spans: []*span.Span{
			{
				SpanId:     "s2",
				Attributes: nil,
			},
		},
	}
	dtoNilAttrs := &parser.AssertionDTO{
		Name: "nil-attrs-size",
		Type: "cel",
		Queries: map[string]any{
			"query1": "size(trace.spans[0].attributes) == 0",
		},
	}
	celNilAttrs, err := assertion.NewCelAssertion(dtoNilAttrs)
	if err != nil {
		t.Fatalf("NewCelAssertion failed: %v", err)
	}
	ok, err = celNilAttrs.Assert(trNoAttrs)
	if !ok || err != nil {
		t.Errorf("Expected nil attributes to be treated as empty map, got ok=%v, err=%v", ok, err)
	}

	// Case 4: Multi-span trace with attribute filtering
	trMulti := &trace.Trace{
		TraceId:    traceID,
		SpanCount:  2,
		ErrorCount: 0,
		Spans: []*span.Span{
			{
				SpanId: "s1",
				Attributes: map[string]any{
					"http.method": "GET",
				},
			},
			{
				SpanId: "s2",
				Attributes: map[string]any{
					"http.method": "POST",
				},
			},
		},
	}
	dtoMulti := &parser.AssertionDTO{
		Name: "multi-span-attrs",
		Type: "cel",
		Queries: map[string]any{
			"query1": "trace.spans.exists(s, s.attributes['http.method'] == 'POST')",
		},
	}
	celMulti, err := assertion.NewCelAssertion(dtoMulti)
	if err != nil {
		t.Fatalf("NewCelAssertion failed: %v", err)
	}
	ok, err = celMulti.Assert(trMulti)
	if !ok || err != nil {
		t.Errorf("Expected multi-span attribute assertion to pass, got ok=%v, err=%v", ok, err)
	}
}
