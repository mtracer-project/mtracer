package dockersetupcommand_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	dockersetupcommand "github.com/mtrace-project/mtrace/setupCommand/docker"
)

func TestNewComposeUpCommand(t *testing.T) {
	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewComposeUpCommand(nil, "/base", context.Background())
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing composePath", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "composeup",
			Args: map[string]any{},
		}
		_, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err == nil || !strings.Contains(err.Error(), "composePath argument is required") {
			t.Errorf("expected error about composePath required, got %v", err)
		}
	})

	t.Run("empty composePath", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "composeup",
			Args: map[string]any{
				"composePath": "",
			},
		}
		_, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err == nil || !strings.Contains(err.Error(), "composePath argument is required") {
			t.Errorf("expected error about composePath required, got %v", err)
		}
	})

	t.Run("wrong type composePath", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "composeup",
			Args: map[string]any{
				"composePath": 12345,
			},
		}
		_, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err == nil || !strings.Contains(err.Error(), "composePath argument is required") {
			t.Errorf("expected error about composePath required, got %v", err)
		}
	})

	t.Run("valid command construction with project name", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "composeup",
			Args: map[string]any{
				"composePath": "docker-compose.yml",
				"projectName": "my-project",
			},
		}
		cmd, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func setupMockDocker(t *testing.T, fail bool) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	logFile := filepath.Join(tmpDir, "args.log")
	scriptPath := filepath.Join(binDir, "docker")

	scriptContent := `#!/bin/sh
echo "$@" >> "` + logFile + `"
if [ "$MTRACE_TEST_FAIL" = "true" ]; then
  echo "mock docker error output" >&2
  exit 1
else
  echo "mock docker success output"
  exit 0
fi
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write mock docker script: %v", err)
	}

	oldPath := os.Getenv("PATH")
	newPath := binDir + string(filepath.ListSeparator) + oldPath
	os.Setenv("PATH", newPath)             // nolint: errcheck
	os.Setenv("MTRACE_TEST_FAIL", "false") // nolint: errcheck
	if fail {
		os.Setenv("MTRACE_TEST_FAIL", "true") // nolint: errcheck
	}

	cleanup := func() {
		os.Setenv("PATH", oldPath)      // nolint: errcheck
		os.Unsetenv("MTRACE_TEST_FAIL") // nolint: errcheck
	}

	return logFile, cleanup
}

func TestComposeUpCommand_Execute(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd: "composeup",
		Args: map[string]any{
			"composePath": "docker-compose.yml",
			"projectName": "my-project",
		},
	}

	t.Run("execute success", func(t *testing.T) {
		logFile, cleanup := setupMockDocker(t, false)
		defer cleanup()

		cmd, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		argsStr := strings.TrimSpace(string(content))
		expectedArgs := "compose -f /base/docker-compose.yml -p my-project up -d --wait"
		if argsStr != expectedArgs {
			t.Errorf("expected docker args %q, got %q", expectedArgs, argsStr)
		}
	})

	t.Run("execute success without project name", func(t *testing.T) {
		logFile, cleanup := setupMockDocker(t, false)
		defer cleanup()

		dtoNoProj := &parser.SetupCommandDTO{
			Cmd: "composeup",
			Args: map[string]any{
				"composePath": "docker-compose.yml",
			},
		}

		cmd, err := dockersetupcommand.NewComposeUpCommand(dtoNoProj, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected execute error: %v", err)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		argsStr := strings.TrimSpace(string(content))
		expectedArgs := "compose -f /base/docker-compose.yml up -d --wait"
		if argsStr != expectedArgs {
			t.Errorf("expected docker args %q, got %q", expectedArgs, argsStr)
		}
	})

	t.Run("execute failure", func(t *testing.T) {
		_, cleanup := setupMockDocker(t, true)
		defer cleanup()

		cmd, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err == nil {
			t.Fatal("expected error from execute, got nil")
		}

		if !strings.Contains(err.Error(), "docker compose up failed") {
			t.Errorf("expected docker compose up failed error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "mock docker error output") {
			t.Errorf("expected error to include command output, got: %v", err)
		}
	})
}

func TestComposeUpCommand_Cleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd: "composeup",
		Args: map[string]any{
			"composePath": "docker-compose.yml",
			"projectName": "my-project",
		},
	}

	t.Run("cleanup success", func(t *testing.T) {
		logFile, cleanup := setupMockDocker(t, false)
		defer cleanup()

		cmd, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("unexpected cleanup error: %v", err)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		argsStr := strings.TrimSpace(string(content))
		expectedArgs := "compose -f /base/docker-compose.yml -p my-project down -v --remove-orphans"
		if argsStr != expectedArgs {
			t.Errorf("expected docker args %q, got %q", expectedArgs, argsStr)
		}
	})

	t.Run("cleanup success without project name", func(t *testing.T) {
		logFile, cleanup := setupMockDocker(t, false)
		defer cleanup()

		dtoNoProj := &parser.SetupCommandDTO{
			Cmd: "composeup",
			Args: map[string]any{
				"composePath": "docker-compose.yml",
			},
		}

		cmd, err := dockersetupcommand.NewComposeUpCommand(dtoNoProj, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("unexpected cleanup error: %v", err)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		argsStr := strings.TrimSpace(string(content))
		expectedArgs := "compose -f /base/docker-compose.yml down -v --remove-orphans"
		if argsStr != expectedArgs {
			t.Errorf("expected docker args %q, got %q", expectedArgs, argsStr)
		}
	})

	t.Run("cleanup failure", func(t *testing.T) {
		_, cleanup := setupMockDocker(t, true)
		defer cleanup()

		cmd, err := dockersetupcommand.NewComposeUpCommand(dto, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Cleanup()
		if err == nil {
			t.Fatal("expected error from cleanup, got nil")
		}

		if !strings.Contains(err.Error(), "docker compose down failed") {
			t.Errorf("expected docker compose down failed error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "mock docker error output") {
			t.Errorf("expected error to include command output, got: %v", err)
		}
	})
}
