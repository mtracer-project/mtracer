package shellsetupcommand_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	setupcommand "github.com/mtracer-project/mtracer/setupCommand/shell"
	testutils "github.com/mtracer-project/mtracer/testUtils"
)

func makeShellDTO(cmd string, cleanup *parser.CleanupCommandDTO) *parser.SetupCommandDTO {
	return &parser.SetupCommandDTO{
		Type:       "shell",
		Cmd:        cmd,
		CleanupCmd: cleanup,
	}
}

func TestNewShellSetupCommand(t *testing.T) {
	t.Run("success with cleanupCmd", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		}

		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("echo 'cmd'", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("failure when cleanupCmd is nil", func(t *testing.T) {
		_, err := setupcommand.NewShellSetupCommand(makeShellDTO("echo 'cmd'", nil), "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "cleanup command is required") {
			t.Errorf("expected error about mandatory cleanupCmd, got: %v", err)
		}
	})
}

func TestNewShellCleanupCommand(t *testing.T) {
	t.Run("nil dto returns nil", func(t *testing.T) {
		cmd := setupcommand.NewShellCleanupCommand(nil, "", context.Background())
		if cmd != nil {
			t.Errorf("expected nil, got %v", cmd)
		}
	})

	t.Run("non-nil dto", func(t *testing.T) {
		dto := &parser.CleanupCommandDTO{Cmd: "echo 1"}
		cmd := setupcommand.NewShellCleanupCommand(dto, "", context.Background())
		if cmd == nil {
			t.Fatal("expected non-nil")
		}
	})
}

func TestShellSetupCommand_Execute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		}
		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("echo 'hello'", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("failed to create command: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Errorf("unexpected execute error: %v", err)
		}
	})

	t.Run("execution with output enabled", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		}
		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("echo 'hello world'", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("failed to create command: %v", err)
		}

		var execErr error
		output := testutils.CaptureStdout(t, func() {
			execErr = cmd.Execute()
		})

		if execErr != nil {
			t.Errorf("unexpected error: %v", execErr)
		}

		if !strings.Contains(output, "Output of command executed") || !strings.Contains(output, "command=\"echo 'hello world'\"") {
			t.Errorf("expected stdout output to contain slog message and command, got: %q", output)
		}
	})

	t.Run("failed execution with invalid command", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		}
		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("nonexistentcommand12345", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("failed to create command: %v", err)
		}

		err = cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to execute setup command") {
			t.Errorf("expected error about command execution failure, got: %v", err)
		}
	})

	t.Run("empty command execution", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		}
		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("  ", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("failed to create command: %v", err)
		}

		err = cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "no command provided") {
			t.Errorf("expected error about no command provided, got: %v", err)
		}
	})

	t.Run("command split syntax error", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		}

		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("echo 'unclosed quote", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("failed to create command: %v", err)
		}

		err = cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "error parsing command") {
			t.Errorf("expected error about parsing command, got: %v", err)
		}
	})
}

func TestShellSetupCommand_Cleanup(t *testing.T) {
	t.Run("successful cleanup execution", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "echo 'cleanup done'",
		}
		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("echo 'cmd'", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("failed to create command: %v", err)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Errorf("unexpected cleanup error: %v", err)
		}
	})

	t.Run("failed cleanup execution with invalid command", func(t *testing.T) {
		cleanup := &parser.CleanupCommandDTO{
			Cmd: "nonexistentcleanup12345",
		}
		cmd, err := setupcommand.NewShellSetupCommand(makeShellDTO("echo 'cmd'", cleanup), "", context.Background())
		if err != nil {
			t.Fatalf("failed to create command: %v", err)
		}

		err = cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to execute cleanup command") {
			t.Errorf("expected error about cleanup failure, got: %v", err)
		}
	})

	t.Run("error when cleanupCmd is nil (direct struct construction)", func(t *testing.T) {
		cmd := &setupcommand.ShellSetupCommand{
			// cmd: "echo 'cmd'",
			// cleanupCmd: nil,
		}

		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "no cleanup command defined") {
			t.Errorf("expected error about no cleanup command defined, got: %v", err)
		}
	})
}
