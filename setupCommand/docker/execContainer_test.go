package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	dockersetupcommand "github.com/mtracer-project/mtracer/setupCommand/docker"
)

type mockContainerCommandExecutor struct {
	calledId  string
	calledCmd string
	returnErr error
}

func (m *mockContainerCommandExecutor) Execute(containerId string, cmd string) error {
	m.calledId = containerId
	m.calledCmd = cmd
	return m.returnErr
}

func TestNewExecContainerCommand(t *testing.T) {
	mockExec := &mockContainerCommandExecutor{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewExecContainerCommand(nil, mockExec)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing cleanupCmd", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"containerId": "test-id", "cmd": "ls"},
		}
		_, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err == nil || !strings.Contains(err.Error(), "cleanup command is required") {
			t.Errorf("expected error about cleanup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"cmd": "ls"},
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'clean'",
			},
		}
		_, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("empty containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"containerId": "", "cmd": "ls"},
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'clean'",
			},
		}
		_, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("missing cmd", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"containerId": "test-id"},
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'clean'",
			},
		}
		_, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err == nil || !strings.Contains(err.Error(), "cmd argument is required") {
			t.Errorf("expected error about cmd required, got %v", err)
		}
	})

	t.Run("empty cmd", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"containerId": "test-id", "cmd": ""},
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'clean'",
			},
		}
		_, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err == nil || !strings.Contains(err.Error(), "cmd argument is required") {
			t.Errorf("expected error about cmd required, got %v", err)
		}
	})

	t.Run("wrong type cmd", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"containerId": "test-id", "cmd": 123},
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'clean'",
			},
		}
		_, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err == nil || !strings.Contains(err.Error(), "cmd argument is required") {
			t.Errorf("expected error about cmd required, got %v", err)
		}
	})

	t.Run("valid construction", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "execcontainer",
			Args: map[string]any{"containerId": "test-id", "cmd": "ls"},
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'clean'",
			},
		}
		cmd, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestNewExecContainerCleanupCommand(t *testing.T) {
	mockExec := &mockContainerCommandExecutor{}

	t.Run("nil DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewExecContainerCleanupCommand(nil, "test-container", mockExec)
		if err == nil || !strings.Contains(err.Error(), "cleanup command DTO is required") {
			t.Errorf("expected error about cleanup DTO required, got %v", err)
		}
	})

	t.Run("empty command in Cmd", func(t *testing.T) {
		dto := &parser.CleanupCommandDTO{
			Cmd: "",
		}
		_, err := dockersetupcommand.NewExecContainerCleanupCommand(dto, "test-container", mockExec)
		if err == nil || !strings.Contains(err.Error(), "cleanup command string is required and cannot be empty") {
			t.Errorf("expected error about empty command, got %v", err)
		}
	})

	t.Run("valid command", func(t *testing.T) {
		dto := &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		}
		cmd, err := dockersetupcommand.NewExecContainerCleanupCommand(dto, "test-container", mockExec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestExecContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd:  "execcontainer",
		Args: map[string]any{"containerId": "test-id", "cmd": "ls"},
		CleanupCmd: &parser.CleanupCommandDTO{
			Cmd: "echo 'clean'",
		},
	}

	t.Run("success execution and cleanup", func(t *testing.T) {
		mockExec := &mockContainerCommandExecutor{}
		cmd, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("unexpected execution error: %v", err)
		}
		if mockExec.calledId != "test-id" || mockExec.calledCmd != "ls" {
			t.Errorf("expected mock executor called with test-id and ls, got %s and %s", mockExec.calledId, mockExec.calledCmd)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("unexpected cleanup error: %v", err)
		}
		if mockExec.calledId != "test-id" || mockExec.calledCmd != "echo 'clean'" {
			t.Errorf("expected mock executor called with test-id and echo 'clean', got %s and %s", mockExec.calledId, mockExec.calledCmd)
		}
	})

	t.Run("execution failure", func(t *testing.T) {
		mockExec := &mockContainerCommandExecutor{
			returnErr: errors.New("exec error"),
		}
		cmd, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to execute command in container") {
			t.Errorf("expected error about execution failure, got %v", err)
		}
	})

	t.Run("cleanup failure", func(t *testing.T) {
		mockExec := &mockContainerCommandExecutor{}
		cmd, err := dockersetupcommand.NewExecContainerCommand(dto, mockExec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mockExec.returnErr = errors.New("cleanup error")
		err = cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to execute cleanup command in container") {
			t.Errorf("expected error about cleanup failure, got %v", err)
		}
	})
}
