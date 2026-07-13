package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	dockersetupcommand "github.com/mtracer-project/mtracer/setupCommand/docker"
)

type mockPacketLossThrottler struct {
	calledCmd          string
	calledNetInterface string
	calledContainerId  string
	throttleErr        error
	unthrottleErr      error
}

func (m *mockPacketLossThrottler) Throttle(cmd string, netInterface string, targetContainerId string) error {
	m.calledCmd = cmd
	m.calledNetInterface = netInterface
	m.calledContainerId = targetContainerId
	return m.throttleErr
}

func (m *mockPacketLossThrottler) Unthrottle(netInterface string, targetContainerId string) error {
	m.calledNetInterface = netInterface
	m.calledContainerId = targetContainerId
	return m.unthrottleErr
}

func TestNewPacketLossContainerCommand(t *testing.T) {
	mockThrottler := &mockPacketLossThrottler{}

	t.Run("nil cmd DTO", func(t *testing.T) {
		_, err := dockersetupcommand.NewPacketLossContainerCommand(nil, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "setup command is required") {
			t.Errorf("expected error about setup command required, got %v", err)
		}
	})

	t.Run("missing containerId", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "packetlosscontainer",
			Args: map[string]any{"loss": "5%"},
		}
		_, err := dockersetupcommand.NewPacketLossContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "containerId argument is required") {
			t.Errorf("expected error about containerId required, got %v", err)
		}
	})

	t.Run("missing loss", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "packetlosscontainer",
			Args: map[string]any{"containerId": "test-id"},
		}
		_, err := dockersetupcommand.NewPacketLossContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "loss argument is required") {
			t.Errorf("expected error about loss required, got %v", err)
		}
	})

	t.Run("invalid loss format", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "packetlosscontainer",
			Args: map[string]any{"containerId": "test-id", "loss": "invalid"},
		}
		_, err := dockersetupcommand.NewPacketLossContainerCommand(dto, mockThrottler)
		if err == nil || !strings.Contains(err.Error(), "invalid loss format") {
			t.Errorf("expected error about invalid loss, got %v", err)
		}
	})

	t.Run("valid construction with percentage suffix", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Cmd:  "packetlosscontainer",
			Args: map[string]any{"containerId": "test-id", "loss": "5.5%"},
		}
		cmd, err := dockersetupcommand.NewPacketLossContainerCommand(dto, mockThrottler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})
}

func TestPacketLossContainerCommand_ExecuteAndCleanup(t *testing.T) {
	dto := &parser.SetupCommandDTO{
		Cmd:  "packetlosscontainer",
		Args: map[string]any{"containerId": "test-id", "loss": "5.5%", "netInterface": "eth3"},
	}

	t.Run("success", func(t *testing.T) {
		mockThrottler := &mockPacketLossThrottler{}
		cmd, err := dockersetupcommand.NewPacketLossContainerCommand(dto, mockThrottler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = cmd.Execute()
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if mockThrottler.calledCmd != "netem loss 5.500000%" || mockThrottler.calledNetInterface != "eth3" || mockThrottler.calledContainerId != "test-id" {
			t.Errorf("unexpected throttle arguments: %v", mockThrottler)
		}

		err = cmd.Cleanup()
		if err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
		if mockThrottler.calledNetInterface != "eth3" || mockThrottler.calledContainerId != "test-id" {
			t.Errorf("unexpected unthrottle arguments: %v", mockThrottler)
		}
	})

	t.Run("throttle error", func(t *testing.T) {
		mockThrottler := &mockPacketLossThrottler{throttleErr: errors.New("throttle error")}
		cmd, _ := dockersetupcommand.NewPacketLossContainerCommand(dto, mockThrottler)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "failed to throttle container") {
			t.Errorf("expected error from execute, got %v", err)
		}
	})

	t.Run("unthrottle error", func(t *testing.T) {
		mockThrottler := &mockPacketLossThrottler{unthrottleErr: errors.New("unthrottle error")}
		cmd, _ := dockersetupcommand.NewPacketLossContainerCommand(dto, mockThrottler)
		err := cmd.Cleanup()
		if err == nil || !strings.Contains(err.Error(), "failed to unthrottle container") {
			t.Errorf("expected error from cleanup, got %v", err)
		}
	})
}
