package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	dockersetupcommand "github.com/mtracer-project/mtracer/setupCommand/docker"
)

type mockPausePauser struct {
	calledId  string
	returnErr error
}

func (m *mockPausePauser) Pause(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

type mockPauseUnpauser struct {
	calledId  string
	returnErr error
}

func (m *mockPauseUnpauser) Unpause(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

func TestNewPauseContainerCommand(t *testing.T) {
	mockPauser := &mockPausePauser{}
	mockUnpauser := &mockPauseUnpauser{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewPauseContainerCommand(nil, mockPauser, mockUnpauser)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "pausecontainer",
			Args: map[string]any{},
		}
		_, err := dockersetupcommand.NewPauseContainerCommand(dto, mockPauser, mockUnpauser)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("empty containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "pausecontainer",
			Args: map[string]any{
				"containerId": "",
			},
		}
		_, err := dockersetupcommand.NewPauseContainerCommand(dto, mockPauser, mockUnpauser)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("wrong type containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "pausecontainer",
			Args: map[string]any{
				"containerId": 12345,
			},
		}
		_, err := dockersetupcommand.NewPauseContainerCommand(dto, mockPauser, mockUnpauser)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("valid command construction", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "pausecontainer",
			Args: map[string]any{
				"containerId": "my-test-container",
			},
		}
		cmd, err := dockersetupcommand.NewPauseContainerCommand(dto, mockPauser, mockUnpauser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestPauseContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd: "pausecontainer",
		Args: map[string]any{
			"containerId": "my-test-container",
		},
	}

	t.Run("execute and cleanup success", func(t *testing.T) {
		mockPauser := &mockPausePauser{}
		mockUnpauser := &mockPauseUnpauser{}
		cmd, err := dockersetupcommand.NewPauseContainerCommand(dto, mockPauser, mockUnpauser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if mockPauser.calledId != "my-test-container" {
			t.Errorf("expected pauser to be called with my-test-container, got %s", mockPauser.calledId)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if mockUnpauser.calledId != "my-test-container" {
			t.Errorf("expected unpauser to be called with my-test-container, got %s", mockUnpauser.calledId)
		}
	})

	t.Run("execute failure", func(t *testing.T) {
		mockPauser := &mockPausePauser{returnErr: errors.New("pause error")}
		mockUnpauser := &mockPauseUnpauser{}
		cmd, _ := dockersetupcommand.NewPauseContainerCommand(dto, mockPauser, mockUnpauser)

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to pause container") {
			t.Errorf("expected error from execute, got %v", err)
		}
	})

	t.Run("cleanup failure", func(t *testing.T) {
		mockPauser := &mockPausePauser{}
		mockUnpauser := &mockPauseUnpauser{returnErr: errors.New("unpause error")}
		cmd, _ := dockersetupcommand.NewPauseContainerCommand(dto, mockPauser, mockUnpauser)

		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to unpause container") {
			t.Errorf("expected error from cleanup, got %v", err)
		}
	})
}
