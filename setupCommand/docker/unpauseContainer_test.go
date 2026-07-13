package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	dockersetupcommand "github.com/mtracer-project/mtracer/setupCommand/docker"
)

type mockUnpauseUnpauser struct {
	calledId  string
	returnErr error
}

func (m *mockUnpauseUnpauser) Unpause(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

type mockUnpausePauser struct {
	calledId  string
	returnErr error
}

func (m *mockUnpausePauser) Pause(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

func TestNewUnpauseContainerCommand(t *testing.T) {
	mockUnpauser := &mockUnpauseUnpauser{}
	mockPauser := &mockUnpausePauser{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewUnpauseContainerCommand(nil, mockUnpauser, mockPauser)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "unpausecontainer",
			Args: map[string]any{},
		}
		_, err := dockersetupcommand.NewUnpauseContainerCommand(dto, mockUnpauser, mockPauser)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("empty containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "unpausecontainer",
			Args: map[string]any{
				"containerId": "",
			},
		}
		_, err := dockersetupcommand.NewUnpauseContainerCommand(dto, mockUnpauser, mockPauser)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("wrong type containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "unpausecontainer",
			Args: map[string]any{
				"containerId": 12345,
			},
		}
		_, err := dockersetupcommand.NewUnpauseContainerCommand(dto, mockUnpauser, mockPauser)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("valid command construction", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "unpausecontainer",
			Args: map[string]any{
				"containerId": "my-test-container",
			},
		}
		cmd, err := dockersetupcommand.NewUnpauseContainerCommand(dto, mockUnpauser, mockPauser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestUnpauseContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd: "unpausecontainer",
		Args: map[string]any{
			"containerId": "my-test-container",
		},
	}

	t.Run("execute and cleanup success", func(t *testing.T) {
		mockUnpauser := &mockUnpauseUnpauser{}
		mockPauser := &mockUnpausePauser{}
		cmd, err := dockersetupcommand.NewUnpauseContainerCommand(dto, mockUnpauser, mockPauser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if mockUnpauser.calledId != "my-test-container" {
			t.Errorf("expected unpauser to be called with my-test-container, got %s", mockUnpauser.calledId)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if mockPauser.calledId != "my-test-container" {
			t.Errorf("expected pauser to be called with my-test-container, got %s", mockPauser.calledId)
		}
	})

	t.Run("execute failure", func(t *testing.T) {
		mockUnpauser := &mockUnpauseUnpauser{returnErr: errors.New("unpause error")}
		mockPauser := &mockUnpausePauser{}
		cmd, _ := dockersetupcommand.NewUnpauseContainerCommand(dto, mockUnpauser, mockPauser)

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to unpause container") {
			t.Errorf("expected error from execute, got %v", err)
		}
	})

	t.Run("cleanup failure", func(t *testing.T) {
		mockUnpauser := &mockUnpauseUnpauser{}
		mockPauser := &mockUnpausePauser{returnErr: errors.New("pause error")}
		cmd, _ := dockersetupcommand.NewUnpauseContainerCommand(dto, mockUnpauser, mockPauser)

		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to pause container") {
			t.Errorf("expected error from cleanup, got %v", err)
		}
	})
}
