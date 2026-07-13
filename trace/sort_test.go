package trace_test

import (
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/span"
	"github.com/mtracer-project/mtracer/trace"
)

func TestSortSpansHierarchically(t *testing.T) {
	now := time.Now()

	// 1. Basic Parent-Child-Grandchild
	spans := []*span.Span{
		{SpanId: "grandchild", ParentId: "child", StartTime: now.Add(2 * time.Second)},
		{SpanId: "child", ParentId: "root", StartTime: now.Add(1 * time.Second)},
		{SpanId: "root", ParentId: "", StartTime: now},
	}

	sorted := trace.SortSpansHierarchically(spans)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(sorted))
	}
	if sorted[0].SpanId != "root" || sorted[1].SpanId != "child" || sorted[2].SpanId != "grandchild" {
		t.Errorf("incorrect ordering: %s -> %s -> %s", sorted[0].SpanId, sorted[1].SpanId, sorted[2].SpanId)
	}

	// 2. Siblings ordered by StartTime
	spans = []*span.Span{
		{SpanId: "child2", ParentId: "root", StartTime: now.Add(2 * time.Second)},
		{SpanId: "child1", ParentId: "root", StartTime: now.Add(1 * time.Second)},
		{SpanId: "root", ParentId: "", StartTime: now},
	}

	sorted = trace.SortSpansHierarchically(spans)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(sorted))
	}
	if sorted[0].SpanId != "root" || sorted[1].SpanId != "child1" || sorted[2].SpanId != "child2" {
		t.Errorf("incorrect sibling ordering: %s -> %s -> %s", sorted[0].SpanId, sorted[1].SpanId, sorted[2].SpanId)
	}

	// 3. Parent missing from list behaves as root
	spans = []*span.Span{
		{SpanId: "child", ParentId: "missing-parent", StartTime: now.Add(1 * time.Second)},
		{SpanId: "grandchild", ParentId: "child", StartTime: now.Add(2 * time.Second)},
	}

	sorted = trace.SortSpansHierarchically(spans)
	if len(sorted) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(sorted))
	}
	if sorted[0].SpanId != "child" || sorted[1].SpanId != "grandchild" {
		t.Errorf("incorrect missing parent ordering: %s -> %s", sorted[0].SpanId, sorted[1].SpanId)
	}

	// 4. Empty/Single span list
	if len(trace.SortSpansHierarchically([]*span.Span{})) != 0 {
		t.Error("expected empty result for empty input")
	}
	single := []*span.Span{{SpanId: "one"}}
	if len(trace.SortSpansHierarchically(single)) != 1 || trace.SortSpansHierarchically(single)[0].SpanId != "one" {
		t.Error("expected same single span returned")
	}
}
