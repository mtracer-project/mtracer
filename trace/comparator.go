package trace

import (
	"fmt"
	"strings"

	"github.com/mtracer-project/mtracer/span"
)

func NewTraceSpansComparator(checker string, ordered bool) TraceSpansComparator {
	switch strings.ToLower(checker) {
	case "strict":
		return &StrictTraceSpansComparator{ordered: ordered}
	case "contains":
		return &ContainsTraceSpansComparator{ordered: ordered}
	case "startswith":
		return &StartsWithTraceSpansComparator{ordered: ordered}
	case "endswith":
		return &EndsWithTraceSpansComparator{ordered: ordered}
	default:
		return nil
	}
}

func (t *Trace) CompareProperties(expectedProperties *ExpectedTraceProperties) (bool, string) {
	if expectedProperties != nil {
		if expectedProperties.maxDuration != nil && t.Duration > *expectedProperties.maxDuration {
			return false, fmt.Sprintf("trace duration exceeds maximum, expected: %s, actual: %s", expectedProperties.maxDuration.String(), t.Duration.String())
		}

		if expectedProperties.minDuration != nil && t.Duration < *expectedProperties.minDuration {
			return false, fmt.Sprintf("trace duration is less than minimum, expected: %s, actual: %s", expectedProperties.minDuration.String(), t.Duration.String())
		}

		if expectedProperties.spanCount != nil && *expectedProperties.spanCount != t.SpanCount {
			return false, fmt.Sprintf("span count does not match, expected: %d, actual: %d", *expectedProperties.spanCount, t.SpanCount)
		}

		if expectedProperties.errorCount != nil && *expectedProperties.errorCount != t.ErrorCount {
			return false, fmt.Sprintf("error count does not match, expected: %d, actual: %d", *expectedProperties.errorCount, t.ErrorCount)
		}
	}

	return true, ""
}

func (t *Trace) Compare(expectedTrace *ExpectedTrace) (bool, string) {
	if expectedTrace == nil {
		return false, "expected trace is nil"
	}

	if len(expectedTrace.spans) == 0 {
		return true, ""
	}

	return expectedTrace.comparator.Compare(expectedTrace.spans, t.Spans)
}

// StrictTraceSpansComparator compares expected and actual spans with strict matching (all fields must match)
type StrictTraceSpansComparator struct {
	ordered bool
}

func (c *StrictTraceSpansComparator) Compare(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string) {
	if len(expected) != len(actual) {
		return false, fmt.Sprintf("span count does not match for strict comparison, expected: %d, actual: %d", len(expected), len(actual))
	}

	if c.ordered {
		return matchOrderedExact(expected, actual)
	}

	return matchUnorderedSubset(expected, actual)
}

// ContainsTraceSpansComparator compares expected and actual spans with contains matching (expected fields must be present in actual spans)
type ContainsTraceSpansComparator struct {
	ordered bool
}

func (c *ContainsTraceSpansComparator) Compare(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string) {
	if len(expected) > len(actual) {
		return false, fmt.Sprintf("expected span count is greater than actual span count for contains comparison, expected: %d, actual: %d", len(expected), len(actual))
	}

	if c.ordered {
		return matchOrderedSubsequence(expected, actual)
	}

	return matchUnorderedSubset(expected, actual)
}

// StartsWithTraceSpansComparator compares expected and actual spans with startsWith matching (expected fields must match the beginning of actual spans)
type StartsWithTraceSpansComparator struct {
	ordered bool
}

func (c *StartsWithTraceSpansComparator) Compare(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string) {
	if len(expected) > len(actual) {
		return false, fmt.Sprintf("expected span count is greater than actual span count for startsWith comparison, expected: %d, actual: %d", len(expected), len(actual))
	}

	segment := actual[:len(expected)]
	if c.ordered {
		return matchOrderedExact(expected, segment)
	}

	return matchUnorderedSubset(expected, segment)
}

// EndsWithTraceSpansComparator compares expected and actual spans with endsWith matching (expected fields must match the end of actual spans)
type EndsWithTraceSpansComparator struct {
	ordered bool
}

func (c *EndsWithTraceSpansComparator) Compare(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string) {
	if len(expected) > len(actual) {
		return false, "expected span count is greater than actual span count"
	}

	offset := len(actual) - len(expected)
	segment := actual[offset:]
	if c.ordered {
		return matchOrderedExact(expected, segment)
	}

	return matchUnorderedSubset(expected, segment)
}

func matchOrderedExact(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string) {
	for i, expectedSpan := range expected {
		actualSpan := actual[i]
		equal, reason := actualSpan.Equal(expectedSpan)
		if !equal {
			return false, fmt.Sprintf("expected span at index %d does not match actual span: %s", i+1, reason)
		}
	}

	return true, ""
}

func matchOrderedSubsequence(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string) {
	index := 0
	for _, actualSpan := range actual {
		equal, _ := actualSpan.Equal(expected[index])
		if equal {
			index++
			if index == len(expected) {
				return true, ""
			}
		}
	}

	return false, "expected ordered subsequence of spans not found in actual spans"
}

func matchUnorderedSubset(expected []*span.ExpectedSpan, actual []*span.Span) (bool, string) {
	used := make([]bool, len(actual))
	for _, expectedSpan := range expected {
		found := false
		for i, actualSpan := range actual {
			if used[i] {
				continue
			}

			equal, _ := actualSpan.Equal(expectedSpan)
			if equal {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Sprintf("expected span %v not found in actual spans", expectedSpan)
		}
	}

	return true, ""
}
