package domain_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/domain"
)

func TestExecuteShellCommand(t *testing.T) { //nolint:gocyclo
	t.Run("successful execution returns exit code 0", func(t *testing.T) {
		exitCode, err := domain.ExecuteShellCommand("echo hello", "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})

	t.Run("empty command returns error", func(t *testing.T) {
		exitCode, err := domain.ExecuteShellCommand("", "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "no command provided") {
			t.Errorf("expected 'no command provided' error, got: %v", err)
		}
		if exitCode != 1 {
			t.Errorf("expected exit code 1, got %d", exitCode)
		}
	})

	t.Run("whitespace-only command returns error", func(t *testing.T) {
		exitCode, err := domain.ExecuteShellCommand("   ", "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "no command provided") {
			t.Errorf("expected 'no command provided' error, got: %v", err)
		}
		if exitCode != 1 {
			t.Errorf("expected exit code 1, got %d", exitCode)
		}
	})

	t.Run("unclosed quote returns parse error", func(t *testing.T) {
		exitCode, err := domain.ExecuteShellCommand("echo 'unclosed", "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "error parsing command") {
			t.Errorf("expected 'error parsing command' error, got: %v", err)
		}
		if exitCode != 1 {
			t.Errorf("expected exit code 1, got %d", exitCode)
		}
	})

	t.Run("non-existent command returns error", func(t *testing.T) {
		_, err := domain.ExecuteShellCommand("nonexistentcommand12345", "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "error executing command") {
			t.Errorf("expected 'error executing command' error, got: %v", err)
		}
	})

	t.Run("command with non-zero exit code returns error", func(t *testing.T) {
		exitCode, err := domain.ExecuteShellCommand("false", "", context.Background())
		if err == nil {
			t.Fatal("expected error for command that exits with non-zero code, got nil")
		}
		if exitCode != 1 {
			t.Errorf("expected exit code 1, got %d", exitCode)
		}
	})

	t.Run("baseDir sets working directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a marker file in the temp dir
		markerFile := filepath.Join(tmpDir, "marker.txt")
		if err := os.WriteFile(markerFile, []byte("found"), 0o644); err != nil { // nolint:gosec
			t.Fatalf("failed to create marker file: %v", err)
		}

		// Use cat to read the marker file via relative path from the baseDir
		exitCode, err := domain.ExecuteShellCommand("cat marker.txt", tmpDir, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})

	t.Run("baseDir with non-existent directory returns error", func(t *testing.T) {
		_, err := domain.ExecuteShellCommand("echo hello", "/nonexistent/directory/path", context.Background())
		if err == nil {
			t.Fatal("expected error for non-existent baseDir, got nil")
		}
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // ensure context is expired

		_, err := domain.ExecuteShellCommand("sleep 10", "", ctx)
		if err == nil {
			t.Fatal("expected error for cancelled context, got nil")
		}
	})

	t.Run("command with arguments", func(t *testing.T) {
		exitCode, err := domain.ExecuteShellCommand("echo 'hello world'", "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})

	t.Run("command is trimmed before execution", func(t *testing.T) {
		exitCode, err := domain.ExecuteShellCommand("  echo hello  ", "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})
}
