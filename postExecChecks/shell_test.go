package postexecchecks_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	postexecchecks "github.com/mtrace-project/mtrace/postExecChecks"
)

func TestNewShellPostExecCheck(t *testing.T) {
	t.Run("success with cmd and default exit code", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check1",
			Type: "shell",
			Args: map[string]any{
				"cmd": "echo hello",
			},
		}
		check, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if check == nil {
			t.Fatal("expected non-nil check")
		}
	})

	t.Run("success with custom expected exit code", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check2",
			Type: "shell",
			Args: map[string]any{
				"cmd":              "echo hello",
				"expectedExitCode": 1,
			},
		}
		check, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if check == nil {
			t.Fatal("expected non-nil check")
		}
	})

	t.Run("nil args returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check3",
			Type: "shell",
			Args: nil,
		}
		_, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "args are required") {
			t.Errorf("expected 'args are required' error, got: %v", err)
		}
	})

	t.Run("missing cmd in args returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check4",
			Type: "shell",
			Args: map[string]any{
				"other": "value",
			},
		}
		_, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "cmd is required") {
			t.Errorf("expected 'cmd is required' error, got: %v", err)
		}
	})

	t.Run("cmd with wrong type returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "check5",
			Type: "shell",
			Args: map[string]any{
				"cmd": 123,
			},
		}
		_, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "cmd is required") {
			t.Errorf("expected 'cmd is required' error, got: %v", err)
		}
	})
}

func TestShellPostExecCheck_Check(t *testing.T) {
	t.Run("successful command returns true", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "shell-check-pass",
			Type: "shell",
			Args: map[string]any{
				"cmd": "echo hello",
			},
		}
		check, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		passed, err := check.Check()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !passed {
			t.Error("expected check to pass")
		}
	})

	t.Run("failing command returns false with error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "shell-check-fail",
			Type: "shell",
			Args: map[string]any{
				"cmd": "false",
			},
		}
		check, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = check.Check()
		if err == nil || !strings.Contains(err.Error(), "check failed for command") {
			t.Errorf("expected 'check failed' error, got: %v", err)
		}
	})

	t.Run("non-existent command returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "shell-check-noexist",
			Type: "shell",
			Args: map[string]any{
				"cmd": "nonexistentcommand12345",
			},
		}
		check, err := postexecchecks.NewShellPostExecCheck(dto, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = check.Check()
		if err == nil {
			t.Error("expected error for non-existent command, got nil")
		}
	})

	t.Run("check uses baseDir", func(t *testing.T) {
		tmpDir := t.TempDir()

		dto := &parser.PostExecCheckDTO{
			Name: "shell-check-basedir",
			Type: "shell",
			Args: map[string]any{
				"cmd": "ls",
			},
		}
		check, err := postexecchecks.NewShellPostExecCheck(dto, tmpDir, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		passed, err := check.Check()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !passed {
			t.Error("expected check to pass with valid baseDir")
		}
	})
}
