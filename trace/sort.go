package trace

import (
	"sort"

	"github.com/mtracer-project/mtracer/span"
)

// SortSpansHierarchically sorts a slice of spans hierarchically: parent first, then children.
// Within siblings, they are sorted by StartTime.
func SortSpansHierarchically(spans []*span.Span) []*span.Span {
	if len(spans) <= 1 {
		return spans
	}

	sort.SliceStable(spans, func(i, j int) bool {
		return spans[i].StartTime.Before(spans[j].StartTime)
	})

	spanMap := make(map[string]*span.Span, len(spans))
	for _, s := range spans {
		if s.SpanId != "" {
			spanMap[s.SpanId] = s
		}
	}

	parentToChildren := make(map[string][]*span.Span)
	var roots []*span.Span

	for _, s := range spans {
		if s.ParentId == "" || spanMap[s.ParentId] == nil {
			roots = append(roots, s)
		} else {
			parentToChildren[s.ParentId] = append(parentToChildren[s.ParentId], s)
		}
	}

	ordered := make([]*span.Span, 0, len(spans))
	visited := make(map[string]bool, len(spans))

	var dfs func(s *span.Span)
	dfs = func(s *span.Span) {
		if visited[s.SpanId] {
			return
		}
		visited[s.SpanId] = true

		ordered = append(ordered, s)

		for _, child := range parentToChildren[s.SpanId] {
			dfs(child)
		}
	}

	for _, root := range roots {
		dfs(root)
	}

	return ordered
}
