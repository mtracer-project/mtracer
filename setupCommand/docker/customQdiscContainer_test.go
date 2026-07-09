package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	dockersetupcommand "github.com/mtrace-project/mtrace/setupCommand/docker"
)

type mockCustomQdiscThrottler struct {
	calledCmd          string
	calledNetInterface string
	calledContainerId  string
	throttleErr        error
	unthrottleErr      error
}

func (m *mockCustomQdiscThrottler) Throttle(cmd string, netInterface string, targetContainerId string) error {
	m.calledCmd = cmd
	m.calledNetInterface = netInterface
	m.calledContainerId = targetContainerId
	return m.throttleErr
}

func (m *mockCustomQdiscThrottler) Unthrottle(netInterface string, targetContainerId string) error {
	m.calledNetInterface = netInterface
	m.calledContainerId = targetContainerId
	return m.unthrottleErr
}

func TestNewCustomQdiscContainerCommand(t *testing.T) {
	mockThrottler := &mockCustomQdiscThrottler{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewCustomQdiscContainerCommand(nil, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "customqdisccontainer",
			Args: map[string]any{"qdiscCmd": "loss 5%"},
		}
		_, err := dockersetupcommand.NewCustomQdiscContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("missing qdiscCmd", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "customqdisccontainer",
			Args: map[string]any{"containerId": "test-id"},
		}
		_, err := dockersetupcommand.NewCustomQdiscContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "qdiscCmd argument is required") {
			t.Errorf("expected error about qdiscCmd required, got %v", err)
		}
	})

	t.Run("valid construction with default interface", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "customqdisccontainer",
			Args: map[string]any{"containerId": "test-id", "qdiscCmd": "netem loss 5%"},
		}
		cmd, err := dockersetupcommand.NewCustomQdiscContainerCommand(dto, mockThrottler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestCustomQdiscContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd:  "customqdisccontainer",
		Args: map[string]any{"containerId": "test-id", "qdiscCmd": "netem loss 5%", "netInterface": "eth1"},
	}

	t.Run("success", func(t *testing.T) {
		mockThrottler := &mockCustomQdiscThrottler{}
		cmd, err := dockersetupcommand.NewCustomQdiscContainerCommand(dto, mockThrottler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if mockThrottler.calledCmd != "netem loss 5%" || mockThrottler.calledNetInterface != "eth1" || mockThrottler.calledContainerId != "test-id" {
			t.Errorf("unexpected throttle arguments: %v", mockThrottler)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if mockThrottler.calledNetInterface != "eth1" || mockThrottler.calledContainerId != "test-id" {
			t.Errorf("unexpected unthrottle arguments: %v", mockThrottler)
		}
	})

	t.Run("throttle error", func(t *testing.T) {
		mockThrottler := &mockCustomQdiscThrottler{throttleErr: errors.New("throttle error")}
		cmd, _ := dockersetupcommand.NewCustomQdiscContainerCommand(dto, mockThrottler)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to throttle container") {
			t.Errorf("expected error from execute, got %v", err)
		}
	})

	t.Run("unthrottle error", func(t *testing.T) {
		mockThrottler := &mockCustomQdiscThrottler{unthrottleErr: errors.New("unthrottle error")}
		cmd, _ := dockersetupcommand.NewCustomQdiscContainerCommand(dto, mockThrottler)
		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to unthrottle container") {
			t.Errorf("expected error from cleanup, got %v", err)
		}
	})
}
