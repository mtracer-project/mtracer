package trace_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/parser"
	"github.com/mtrace-project/mtrace/span"
	"github.com/mtrace-project/mtrace/trace"
)

func TestNewTraceSpansComparator(t *testing.T) {
	testCases := []struct {
		checker  string
		ordered  bool
		wantType string
	}{
		{"strict", true, "*trace.StrictTraceSpansComparator"},
		{"contains", false, "*trace.ContainsTraceSpansComparator"},
		{"startswith", true, "*trace.StartsWithTraceSpansComparator"},
		{"endswith", false, "*trace.EndsWithTraceSpansComparator"},
		{"unknown", true, "nil"},
	}

	for _, tc := range testCases {
		t.Run(tc.checker, func(t *testing.T) {
			comp := trace.NewTraceSpansComparator(tc.checker, tc.ordered)
			if tc.wantType == "nil" {
				if comp != nil {
					t.Errorf("Expected nil comparator for checker %q, got %T", tc.checker, comp)
				}
			} else {
				if comp == nil {
					t.Fatalf("Expected non-nil comparator for checker %q", tc.checker)
				}
				gotType := fmt.Sprintf("%T", comp)
				if gotType != tc.wantType {
					t.Errorf("Expected comparator type %q, got %q", tc.wantType, gotType)
				}
			}
		})
	}
}

func TestTraceCompare_Validations(t *testing.T) {
	orderedFalse := false
	checkerStrict := "strict"

	// Case 1: nil expected trace
	tr := &trace.Trace{}
	ok, reason := tr.Compare(nil)
	if ok || reason != "expected trace is nil" {
		t.Errorf("Expected Compare to fail with nil expected, got ok=%v, reason=%q", ok, reason)
	}

	// Case 2: Duration exceeds max
	maxDurVal := domain.Duration(10 * time.Millisecond)
	expectedMax := trace.NewExpectedTraceProperties(&parser.ExpectedTracePropertiesDTO{
		MaxDuration: &maxDurVal,
	})
	durVal := 50 * time.Millisecond
	trExceeds := &trace.Trace{
		Duration: durVal,
	}
	ok, reason = trExceeds.CompareProperties(expectedMax)
	if ok || !strings.Contains(reason, "trace duration exceeds maximum") {
		t.Errorf("Expected duration exceeds max mismatch, got ok=%v, reason=%q", ok, reason)
	}

	// Case 3: Duration below min
	minDurVal := domain.Duration(100 * time.Millisecond)
	expectedMin := trace.NewExpectedTraceProperties(&parser.ExpectedTracePropertiesDTO{
		MinDuration: &minDurVal,
	})
	trBelow := &trace.Trace{
		Duration: durVal,
	}
	ok, reason = trBelow.CompareProperties(expectedMin)
	if ok || !strings.Contains(reason, "trace duration is less than minimum") {
		t.Errorf("Expected duration less than min mismatch, got ok=%v, reason=%q", ok, reason)
	}

	// Case 4: spanCount mismatch
	cnt := 3
	expectedCnt := trace.NewExpectedTraceProperties(&parser.ExpectedTracePropertiesDTO{
		SpanCount: &cnt,
	})
	trBadCnt := &trace.Trace{
		SpanCount: 5,
	}
	ok, reason = trBadCnt.CompareProperties(expectedCnt)
	if ok || !strings.Contains(reason, "span count does not match") {
		t.Errorf("Expected spanCount mismatch, got ok=%v, reason=%q", ok, reason)
	}

	// Case 5: errorCount mismatch
	errCnt := 1
	expectedErrCnt := trace.NewExpectedTraceProperties(&parser.ExpectedTracePropertiesDTO{
		ErrorCount: &errCnt,
	})
	trBadErr := &trace.Trace{
		ErrorCount: 3,
	}
	ok, reason = trBadErr.CompareProperties(expectedErrCnt)
	if ok || !strings.Contains(reason, "error count does not match") {
		t.Errorf("Expected errorCount mismatch, got ok=%v, reason=%q", ok, reason)
	}

	// Case 5.5: nil expected properties
	ok, reason = trBadErr.CompareProperties(nil)
	if !ok || reason != "" {
		t.Errorf("Expected CompareProperties to succeed with nil expected, got ok=%v, reason=%q", ok, reason)
	}

	// Case 6: Empty expected spans
	expectedEmpty := trace.NewExpectedTrace(&parser.ExpectedTraceDTO{
		Ordered: &orderedFalse,
		Checker: &checkerStrict,
	})
	trEmptySpans := &trace.Trace{
		SpanCount: 0,
	}
	ok, reason = trEmptySpans.Compare(expectedEmpty)
	if !ok || reason != "" {
		t.Errorf("Expected true for empty expected spans, got ok=%v, reason=%q", ok, reason)
	}
}

func TestNewExpectedTraceProperties_Nil(t *testing.T) {
	res := trace.NewExpectedTraceProperties(nil)
	if res != nil {
		t.Errorf("Expected nil for nil ExpectedTracePropertiesDTO, got %v", res)
	}
}

func TestStrictTraceSpansComparator(t *testing.T) {
	opA := "op-a"
	opB := "op-b"
	orderedTrue := true
	orderedFalse := false
	checkerStrict := "strict"

	expectedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedTrue,
		Checker: &checkerStrict,
	}

	expectedUnorderedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedFalse,
		Checker: &checkerStrict,
	}

	expectedOrdered := trace.NewExpectedTrace(expectedDTO)
	expectedUnordered := trace.NewExpectedTrace(expectedUnorderedDTO)

	actualSpans := make([]*span.Span, 0, 3)
	actualSpans = append(
		actualSpans,
		&span.Span{ServiceName: "service-a", OperationName: "op-a"},
		&span.Span{ServiceName: "service-b", OperationName: "op-b"},
	)

	actualSpansReversed := []*span.Span{
		{ServiceName: "service-b", OperationName: "op-b"},
		{ServiceName: "service-a", OperationName: "op-a"},
	}

	// Case 1: length mismatch
	trBadLen := &trace.Trace{
		SpanCount: 3,
		Spans:     append(actualSpans, &span.Span{ServiceName: "service-c"}),
	}
	ok, reason := trBadLen.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "span count does not match for strict comparison") {
		t.Errorf("Expected length mismatch failure, got ok=%v, reason=%q", ok, reason)
	}

	// Case 2: Ordered match success
	trMatch := &trace.Trace{
		SpanCount: 2,
		Spans:     actualSpans,
	}
	ok, reason = trMatch.Compare(expectedOrdered)
	if !ok {
		t.Errorf("Expected ordered strict comparison to match, got reason: %s", reason)
	}

	// Case 3: Ordered match mismatch (wrong order)
	trReversed := &trace.Trace{
		SpanCount: 2,
		Spans:     actualSpansReversed,
	}
	ok, reason = trReversed.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "expected span at index 1 does not match") {
		t.Errorf("Expected ordered strict failure for reversed spans, got ok=%v, reason=%q", ok, reason)
	}

	// Case 4: Unordered match success (reversed order matches unordered)
	ok, reason = trReversed.Compare(expectedUnordered)
	if !ok {
		t.Errorf("Expected unordered strict comparison to match reversed order, got reason: %s", reason)
	}

	// Case 5: Unordered mismatch (missing span)
	actualSpansBad := []*span.Span{
		{ServiceName: "service-a", OperationName: "op-a"},
		{ServiceName: "service-c", OperationName: "op-c"},
	}
	trBad := &trace.Trace{
		SpanCount: 2,
		Spans:     actualSpansBad,
	}
	ok, reason = trBad.Compare(expectedUnordered)
	if ok || !strings.Contains(reason, "not found in actual spans") {
		t.Errorf("Expected unordered strict failure for missing span, got ok=%v, reason=%q", ok, reason)
	}
}

func TestContainsTraceSpansComparator(t *testing.T) {
	opA := "op-a"
	opB := "op-b"
	orderedTrue := true
	orderedFalse := false
	checkerContains := "contains"

	expectedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedTrue,
		Checker: &checkerContains,
	}

	expectedUnorderedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedFalse,
		Checker: &checkerContains,
	}

	expectedOrdered := trace.NewExpectedTrace(expectedDTO)
	expectedUnordered := trace.NewExpectedTrace(expectedUnorderedDTO)

	// Case 1: expected > actual span count
	trTooFewSpans := &trace.Trace{
		SpanCount: 1,
		Spans: []*span.Span{
			{ServiceName: "service-a", OperationName: "op-a"},
		},
	}
	ok, reason := trTooFewSpans.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "expected span count is greater than actual span count") {
		t.Errorf("Expected count error, got ok=%v, reason=%q", ok, reason)
	}

	// Case 2: Ordered subsequence success
	trMatchSub := &trace.Trace{
		SpanCount: 4,
		Spans: []*span.Span{
			{ServiceName: "service-x", OperationName: "op-x"},
			{ServiceName: "service-a", OperationName: "op-a"}, // Matches 1st expected
			{ServiceName: "service-y", OperationName: "op-y"},
			{ServiceName: "service-b", OperationName: "op-b"}, // Matches 2nd expected
		},
	}
	ok, reason = trMatchSub.Compare(expectedOrdered)
	if !ok {
		t.Errorf("Expected contains ordered match, got reason: %s", reason)
	}

	// Case 3: Ordered subsequence mismatch (wrong order in subsequence)
	trBadSub := &trace.Trace{
		SpanCount: 4,
		Spans: []*span.Span{
			{ServiceName: "service-b", OperationName: "op-b"}, // Matches 2nd expected first
			{ServiceName: "service-a", OperationName: "op-a"}, // Matches 1st expected second
		},
	}
	ok, reason = trBadSub.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "expected ordered subsequence of spans not found") {
		t.Errorf("Expected ordered contains failure, got ok=%v, reason=%q", ok, reason)
	}

	// Case 4: Unordered contains success
	ok, reason = trBadSub.Compare(expectedUnordered)
	if !ok {
		t.Errorf("Expected unordered contains success, got reason: %s", reason)
	}
}

func TestStartsWithTraceSpansComparator(t *testing.T) {
	opA := "op-a"
	opB := "op-b"
	orderedTrue := true
	orderedFalse := false
	checkerStarts := "startswith"

	expectedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedTrue,
		Checker: &checkerStarts,
	}

	expectedUnorderedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedFalse,
		Checker: &checkerStarts,
	}

	expectedOrdered := trace.NewExpectedTrace(expectedDTO)
	expectedUnordered := trace.NewExpectedTrace(expectedUnorderedDTO)

	// Case 1: count error
	trTooFew := &trace.Trace{
		SpanCount: 1,
		Spans: []*span.Span{
			{ServiceName: "service-a"},
		},
	}
	ok, reason := trTooFew.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "expected span count is greater than actual span count") {
		t.Errorf("Expected count error, got ok=%v, reason=%q", ok, reason)
	}

	// Case 2: Ordered startsWith success
	trMatch := &trace.Trace{
		SpanCount: 3,
		Spans: []*span.Span{
			{ServiceName: "service-a", OperationName: "op-a"}, // Matches 1st expected
			{ServiceName: "service-b", OperationName: "op-b"}, // Matches 2nd expected
			{ServiceName: "service-c", OperationName: "op-c"},
		},
	}
	ok, reason = trMatch.Compare(expectedOrdered)
	if !ok {
		t.Errorf("Expected ordered startsWith match, got reason: %s", reason)
	}

	// Case 3: Ordered startsWith mismatch
	trMismatch := &trace.Trace{
		SpanCount: 3,
		Spans: []*span.Span{
			{ServiceName: "service-a", OperationName: "op-a"},
			{ServiceName: "service-c", OperationName: "op-c"}, // Mismatch
			{ServiceName: "service-b", OperationName: "op-b"},
		},
	}
	ok, reason = trMismatch.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "does not match actual span") {
		t.Errorf("Expected ordered startsWith failure, got ok=%v, reason=%q", ok, reason)
	}

	// Case 4: Unordered startsWith success
	trUnorderedMatch := &trace.Trace{
		SpanCount: 3,
		Spans: []*span.Span{
			{ServiceName: "service-b", OperationName: "op-b"}, // reversed but first two elements
			{ServiceName: "service-a", OperationName: "op-a"},
			{ServiceName: "service-c", OperationName: "op-c"},
		},
	}
	ok, reason = trUnorderedMatch.Compare(expectedUnordered)
	if !ok {
		t.Errorf("Expected unordered startsWith match, got reason: %s", reason)
	}
}

func TestEndsWithTraceSpansComparator(t *testing.T) {
	opA := "op-a"
	opB := "op-b"
	orderedTrue := true
	orderedFalse := false
	checkerEnds := "endswith"

	expectedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedTrue,
		Checker: &checkerEnds,
	}

	expectedUnorderedDTO := &parser.ExpectedTraceDTO{
		Spans: []*parser.ExpectedSpanDTO{
			{SpanDTO: parser.SpanDTO{ServiceName: "service-a", OperationName: &opA}},
			{SpanDTO: parser.SpanDTO{ServiceName: "service-b", OperationName: &opB}},
		},
		Ordered: &orderedFalse,
		Checker: &checkerEnds,
	}

	expectedOrdered := trace.NewExpectedTrace(expectedDTO)
	expectedUnordered := trace.NewExpectedTrace(expectedUnorderedDTO)

	// Case 1: count error
	trTooFew := &trace.Trace{
		SpanCount: 1,
		Spans: []*span.Span{
			{ServiceName: "service-a"},
		},
	}
	ok, reason := trTooFew.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "expected span count is greater than actual span count") {
		t.Errorf("Expected count error, got ok=%v, reason=%q", ok, reason)
	}

	// Case 2: Ordered endsWith success
	trMatch := &trace.Trace{
		SpanCount: 3,
		Spans: []*span.Span{
			{ServiceName: "service-c", OperationName: "op-c"},
			{ServiceName: "service-a", OperationName: "op-a"}, // Matches 1st expected
			{ServiceName: "service-b", OperationName: "op-b"}, // Matches 2nd expected
		},
	}
	ok, reason = trMatch.Compare(expectedOrdered)
	if !ok {
		t.Errorf("Expected ordered endsWith match, got reason: %s", reason)
	}

	// Case 3: Ordered endsWith mismatch
	trMismatch := &trace.Trace{
		SpanCount: 3,
		Spans: []*span.Span{
			{ServiceName: "service-a", OperationName: "op-a"},
			{ServiceName: "service-c", OperationName: "op-c"}, // Mismatch at end
			{ServiceName: "service-b", OperationName: "op-b"},
		},
	}
	ok, reason = trMismatch.Compare(expectedOrdered)
	if ok || !strings.Contains(reason, "does not match actual span") {
		t.Errorf("Expected ordered endsWith failure, got ok=%v, reason=%q", ok, reason)
	}

	// Case 4: Unordered endsWith success
	trUnorderedMatch := &trace.Trace{
		SpanCount: 3,
		Spans: []*span.Span{
			{ServiceName: "service-c", OperationName: "op-c"},
			{ServiceName: "service-b", OperationName: "op-b"}, // reversed end elements
			{ServiceName: "service-a", OperationName: "op-a"},
		},
	}
	ok, reason = trUnorderedMatch.Compare(expectedUnordered)
	if !ok {
		t.Errorf("Expected unordered endsWith match, got reason: %s", reason)
	}
}
