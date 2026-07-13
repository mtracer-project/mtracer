package postexecchecks_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	postexecchecks "github.com/mtracer-project/mtracer/postExecChecks"
)

func TestNewPostExecCheck(t *testing.T) {
	t.Run("shell type", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "shell-factory",
			Type: "shell",
			Args: map[string]any{
				"cmd": "echo hello",
			},
		}
		check, err := postexecchecks.NewPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if check == nil {
			t.Fatal("expected non-nil check")
		}
	})

	t.Run("sql type", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-factory",
			Type: "sql",
			Args: map[string]any{
				"query": "SELECT 1 = 1",
				"dsn":   "test-dsn",
			},
		}
		check, err := postexecchecks.NewPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if check == nil {
			t.Fatal("expected non-nil check")
		}
	})

	t.Run("case insensitive type matching", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "shell-upper",
			Type: "SHELL",
			Args: map[string]any{
				"cmd": "echo hello",
			},
		}
		check, err := postexecchecks.NewPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if check == nil {
			t.Fatal("expected non-nil check")
		}
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "unsupported",
			Type: "invalid-type",
			Args: map[string]any{},
		}
		_, err := postexecchecks.NewPostExecCheck(dto, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "unsupported post exec check type") {
			t.Errorf("expected 'unsupported post exec check type' error, got: %v", err)
		}
	})

	t.Run("shell with invalid args propagates error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "shell-bad",
			Type: "shell",
			Args: nil,
		}
		_, err := postexecchecks.NewPostExecCheck(dto, "", context.Background())
		if err == nil {
			t.Error("expected error for shell with nil args, got nil")
		}
	})

	t.Run("sql with invalid args propagates error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-bad",
			Type: "sql",
			Args: nil,
		}
		_, err := postexecchecks.NewPostExecCheck(dto, "", context.Background())
		if err == nil {
			t.Error("expected error for sql with nil args, got nil")
		}
	})
}

func TestNewPostExecChecks(t *testing.T) {
	t.Run("nil dtos returns nil slice", func(t *testing.T) {
		checks, err := postexecchecks.NewPostExecChecks(nil, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if checks != nil {
			t.Errorf("expected nil, got %v", checks)
		}
	})

	t.Run("empty dtos returns nil slice", func(t *testing.T) {
		checks, err := postexecchecks.NewPostExecChecks([]*parser.PostExecCheckDTO{}, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if checks != nil {
			t.Errorf("expected nil, got %v", checks)
		}
	})

	t.Run("multiple valid checks", func(t *testing.T) {
		dtos := []*parser.PostExecCheckDTO{
			{
				Name: "check-1",
				Type: "shell",
				Args: map[string]any{"cmd": "echo 1"},
			},
			{
				Name: "check-2",
				Type: "shell",
				Args: map[string]any{"cmd": "echo 2"},
			},
		}
		checks, err := postexecchecks.NewPostExecChecks(dtos, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(checks) != 2 {
			t.Errorf("expected 2 checks, got %d", len(checks))
		}
	})

	t.Run("error in one check propagates", func(t *testing.T) {
		dtos := []*parser.PostExecCheckDTO{
			{
				Name: "check-ok",
				Type: "shell",
				Args: map[string]any{"cmd": "echo 1"},
			},
			{
				Name: "check-bad",
				Type: "invalid-type",
				Args: map[string]any{},
			},
		}
		_, err := postexecchecks.NewPostExecChecks(dtos, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "error creating post exec check") {
			t.Errorf("expected 'error creating post exec check' error, got: %v", err)
		}
	})
}
