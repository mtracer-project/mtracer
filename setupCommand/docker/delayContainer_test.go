package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	dockersetupcommand "github.com/mtrace-project/mtrace/setupCommand/docker"
)

type mockDelayThrottler struct {
	calledCmd          string
	calledNetInterface string
	calledContainerId  string
	throttleErr        error
	unthrottleErr      error
}

func (m *mockDelayThrottler) Throttle(cmd string, netInterface string, targetContainerId string) error {
	m.calledCmd = cmd
	m.calledNetInterface = netInterface
	m.calledContainerId = targetContainerId
	return m.throttleErr
}

func (m *mockDelayThrottler) Unthrottle(netInterface string, targetContainerId string) error {
	m.calledNetInterface = netInterface
	m.calledContainerId = targetContainerId
	return m.unthrottleErr
}

func TestNewDelayContainerCommand(t *testing.T) {
	mockThrottler := &mockDelayThrottler{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewDelayContainerCommand(nil, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "delaycontainer",
			Args: map[string]any{"delay": "100ms"},
		}
		_, err := dockersetupcommand.NewDelayContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("missing delay", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "delaycontainer",
			Args: map[string]any{"containerId": "test-id"},
		}
		_, err := dockersetupcommand.NewDelayContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "delay argument is required") {
			t.Errorf("expected error about delay required, got %v", err)
		}
	})

	t.Run("invalid delay format", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "delaycontainer",
			Args: map[string]any{"containerId": "test-id", "delay": "invalid"},
		}
		_, err := dockersetupcommand.NewDelayContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "invalid delay format") {
			t.Errorf("expected error about invalid delay, got %v", err)
		}
	})

	t.Run("valid construction", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "delaycontainer",
			Args: map[string]any{"containerId": "test-id", "delay": "100ms"},
		}
		cmd, err := dockersetupcommand.NewDelayContainerCommand(dto, mockThrottler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestDelayContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd:  "delaycontainer",
		Args: map[string]any{"containerId": "test-id", "delay": "100ms", "netInterface": "eth2"},
	}

	t.Run("success", func(t *testing.T) {
		mockThrottler := &mockDelayThrottler{}
		cmd, err := dockersetupcommand.NewDelayContainerCommand(dto, mockThrottler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if mockThrottler.calledCmd != "netem delay 100ms" || mockThrottler.calledNetInterface != "eth2" || mockThrottler.calledContainerId != "test-id" {
			t.Errorf("unexpected throttle arguments: %v", mockThrottler)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if mockThrottler.calledNetInterface != "eth2" || mockThrottler.calledContainerId != "test-id" {
			t.Errorf("unexpected unthrottle arguments: %v", mockThrottler)
		}
	})

	t.Run("throttle error", func(t *testing.T) {
		mockThrottler := &mockDelayThrottler{throttleErr: errors.New("throttle error")}
		cmd, _ := dockersetupcommand.NewDelayContainerCommand(dto, mockThrottler)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to throttle container") {
			t.Errorf("expected error from execute, got %v", err)
		}
	})

	t.Run("unthrottle error", func(t *testing.T) {
		mockThrottler := &mockDelayThrottler{unthrottleErr: errors.New("unthrottle error")}
		cmd, _ := dockersetupcommand.NewDelayContainerCommand(dto, mockThrottler)
		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to unthrottle container") {
			t.Errorf("expected error from cleanup, got %v", err)
		}
	})
}
