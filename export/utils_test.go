package export

import (
	"testing"

	"github.com/mtrace-project/mtrace/test"
)

func TestFormatDetails(t *testing.T) {
	t.Run("nil result", func(t *testing.T) {
		res := formatDetails(nil)
		if res != "" {
			t.Errorf("expected empty string, got %q", res)
		}
	})

	t.Run("empty args", func(t *testing.T) {
		res := formatDetails(&test.TestResult{Args: nil})
		if res != "" {
			t.Errorf("expected empty string, got %q", res)
		}
	})

	t.Run("even number of args", func(t *testing.T) {
		res := formatDetails(&test.TestResult{
			Args: []any{"key1", "val1", "key2", 42},
		})
		expected := "key1: val1 | key2: 42"
		if res != expected {
			t.Errorf("expected %q, got %q", expected, res)
		}
	})

	t.Run("odd number of args", func(t *testing.T) {
		res := formatDetails(&test.TestResult{
			Args: []any{"key1", "val1", "key2"},
		})
		expected := "key1: val1"
		if res != expected {
			t.Errorf("expected %q, got %q", expected, res)
		}
	})
}
