package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	dockersetupcommand "github.com/mtrace-project/mtrace/setupCommand/docker"
)

type mockStartStarter struct {
	calledId  string
	returnErr error
}

func (m *mockStartStarter) Start(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

type mockStartStopper struct {
	calledId  string
	returnErr error
}

func (m *mockStartStopper) Stop(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

func TestNewStartContainerCommand(t *testing.T) {
	mockStarter := &mockStartStarter{}
	mockStopper := &mockStartStopper{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewStartContainerCommand(nil, mockStarter, mockStopper)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "startcontainer",
			Args: map[string]any{},
		}
		_, err := dockersetupcommand.NewStartContainerCommand(dto, mockStarter, mockStopper)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("empty containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "startcontainer",
			Args: map[string]any{
				"containerId": "",
			},
		}
		_, err := dockersetupcommand.NewStartContainerCommand(dto, mockStarter, mockStopper)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("wrong type containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "startcontainer",
			Args: map[string]any{
				"containerId": 12345,
			},
		}
		_, err := dockersetupcommand.NewStartContainerCommand(dto, mockStarter, mockStopper)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("valid command construction", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "startcontainer",
			Args: map[string]any{
				"containerId": "my-test-container",
			},
		}
		cmd, err := dockersetupcommand.NewStartContainerCommand(dto, mockStarter, mockStopper)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestStartContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd: "startcontainer",
		Args: map[string]any{
			"containerId": "my-test-container",
		},
	}

	t.Run("execute and cleanup success", func(t *testing.T) {
		mockStarter := &mockStartStarter{}
		mockStopper := &mockStartStopper{}
		cmd, err := dockersetupcommand.NewStartContainerCommand(dto, mockStarter, mockStopper)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if mockStarter.calledId != "my-test-container" {
			t.Errorf("expected starter to be called with my-test-container, got %s", mockStarter.calledId)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if mockStopper.calledId != "my-test-container" {
			t.Errorf("expected stopper to be called with my-test-container, got %s", mockStopper.calledId)
		}
	})

	t.Run("execute failure", func(t *testing.T) {
		mockStarter := &mockStartStarter{returnErr: errors.New("start error")}
		mockStopper := &mockStartStopper{}
		cmd, _ := dockersetupcommand.NewStartContainerCommand(dto, mockStarter, mockStopper)

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to start container") {
			t.Errorf("expected error from execute, got %v", err)
		}
	})

	t.Run("cleanup failure", func(t *testing.T) {
		mockStarter := &mockStartStarter{}
		mockStopper := &mockStartStopper{returnErr: errors.New("stop error")}
		cmd, _ := dockersetupcommand.NewStartContainerCommand(dto, mockStarter, mockStopper)

		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to stop container") {
			t.Errorf("expected error from cleanup, got %v", err)
		}
	})
}
