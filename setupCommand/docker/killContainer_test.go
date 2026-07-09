package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	dockersetupcommand "github.com/mtrace-project/mtrace/setupCommand/docker"
)

type mockKillKiller struct {
	calledId  string
	returnErr error
}

func (m *mockKillKiller) Kill(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

type mockKillStarter struct {
	calledId  string
	returnErr error
}

func (m *mockKillStarter) Start(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

func TestNewKillContainerCommand(t *testing.T) {
	mockKiller := &mockKillKiller{}
	mockStarter := &mockKillStarter{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewKillContainerCommand(nil, mockKiller, mockStarter)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "killcontainer",
			Args: map[string]any{},
		}
		_, err := dockersetupcommand.NewKillContainerCommand(dto, mockKiller, mockStarter)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("empty containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "killcontainer",
			Args: map[string]any{
				"containerId": "",
			},
		}
		_, err := dockersetupcommand.NewKillContainerCommand(dto, mockKiller, mockStarter)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("wrong type containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "killcontainer",
			Args: map[string]any{
				"containerId": 12345,
			},
		}
		_, err := dockersetupcommand.NewKillContainerCommand(dto, mockKiller, mockStarter)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("valid command construction", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd: "killcontainer",
			Args: map[string]any{
				"containerId": "my-test-container",
			},
		}
		cmd, err := dockersetupcommand.NewKillContainerCommand(dto, mockKiller, mockStarter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestKillContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd: "killcontainer",
		Args: map[string]any{
			"containerId": "my-test-container",
		},
	}

	t.Run("execute and cleanup success", func(t *testing.T) {
		mockKiller := &mockKillKiller{}
		mockStarter := &mockKillStarter{}
		cmd, err := dockersetupcommand.NewKillContainerCommand(dto, mockKiller, mockStarter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if mockKiller.calledId != "my-test-container" {
			t.Errorf("expected killer to be called with my-test-container, got %s", mockKiller.calledId)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if mockStarter.calledId != "my-test-container" {
			t.Errorf("expected starter to be called with my-test-container, got %s", mockStarter.calledId)
		}
	})

	t.Run("execute failure", func(t *testing.T) {
		mockKiller := &mockKillKiller{returnErr: errors.New("kill error")}
		mockStarter := &mockKillStarter{}
		cmd, _ := dockersetupcommand.NewKillContainerCommand(dto, mockKiller, mockStarter)

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to kill container") {
			t.Errorf("expected error from execute, got %v", err)
		}
	})

	t.Run("cleanup failure", func(t *testing.T) {
		mockKiller := &mockKillKiller{}
		mockStarter := &mockKillStarter{returnErr: errors.New("start error")}
		cmd, _ := dockersetupcommand.NewKillContainerCommand(dto, mockKiller, mockStarter)

		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to restart container") {
			t.Errorf("expected error from cleanup, got %v", err)
		}
	})
}
