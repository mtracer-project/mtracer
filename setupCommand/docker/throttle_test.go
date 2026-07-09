package dockersetupcommand_test

import (
	"errors"
	"strings"
	"testing"

	dockersetupcommand "github.com/mtrace-project/mtrace/setupCommand/docker"
)

type mockThrottleBuilder struct {
	calledTargetId     string
	calledNetInterface string
	calledCmd          string
	returnId           string
	returnErr          error
}

func (m *mockThrottleBuilder) Build(targetContainerId string, netInterface string, cmd string) (string, error) {
	m.calledTargetId = targetContainerId
	m.calledNetInterface = netInterface
	m.calledCmd = cmd
	return m.returnId, m.returnErr
}

type mockThrottleStarter struct {
	calledId  string
	returnErr error
}

func (m *mockThrottleStarter) Start(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

type mockThrottleStopper struct {
	calledId  string
	returnErr error
}

func (m *mockThrottleStopper) Stop(containerId string) error {
	m.calledId = containerId
	return m.returnErr
}

type mockThrottleExecutor struct {
	calledId  string
	calledCmd string
	returnErr error
}

func (m *mockThrottleExecutor) Execute(containerId string, cmd string) error {
	m.calledId = containerId
	m.calledCmd = cmd
	return m.returnErr
}

func TestDockerContainerThrottler_Throttle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		builder := &mockThrottleBuilder{returnId: "helper-123"}
		starter := &mockThrottleStarter{}
		stopper := &mockThrottleStopper{}
		executor := &mockThrottleExecutor{}

		throttler := dockersetupcommand.NewDockerContainerThrottler(builder, starter, stopper, executor)
		err := throttler.Throttle("netem loss 5%", "eth0", "target-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if builder.calledTargetId != "target-456" || builder.calledNetInterface != "eth0" || builder.calledCmd != "netem loss 5%" {
			t.Errorf("unexpected builder calls: %v", builder)
		}
		if starter.calledId != "helper-123" {
			t.Errorf("expected helper container to be started, got %s", starter.calledId)
		}
	})

	t.Run("builder error", func(t *testing.T) {
		builder := &mockThrottleBuilder{returnErr: errors.New("build error")}
		starter := &mockThrottleStarter{}
		stopper := &mockThrottleStopper{}
		executor := &mockThrottleExecutor{}

		throttler := dockersetupcommand.NewDockerContainerThrottler(builder, starter, stopper, executor)
		err := throttler.Throttle("netem loss 5%", "eth0", "target-456")
		if err == nil || !strings.Contains(err.Error(), "failed to build helper container") {
			t.Errorf("expected builder error, got %v", err)
		}
	})

	t.Run("starter error", func(t *testing.T) {
		builder := &mockThrottleBuilder{returnId: "helper-123"}
		starter := &mockThrottleStarter{returnErr: errors.New("start error")}
		stopper := &mockThrottleStopper{}
		executor := &mockThrottleExecutor{}

		throttler := dockersetupcommand.NewDockerContainerThrottler(builder, starter, stopper, executor)
		err := throttler.Throttle("netem loss 5%", "eth0", "target-456")
		if err == nil || !strings.Contains(err.Error(), "failed to start helper container") {
			t.Errorf("expected starter error, got %v", err)
		}
	})
}

func TestDockerContainerThrottler_Unthrottle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		builder := &mockThrottleBuilder{returnId: "helper-123"}
		starter := &mockThrottleStarter{}
		stopper := &mockThrottleStopper{}
		executor := &mockThrottleExecutor{}

		throttler := dockersetupcommand.NewDockerContainerThrottler(builder, starter, stopper, executor)
		// Set helper container ID by calling Throttle first
		_ = throttler.Throttle("netem loss 5%", "eth0", "target-456")

		err := throttler.Unthrottle("eth0", "target-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if executor.calledId != "helper-123" || executor.calledCmd != "tc qdisc del dev eth0 root" {
			t.Errorf("unexpected executor call: %v", executor)
		}
		if stopper.calledId != "helper-123" {
			t.Errorf("expected helper container to be stopped, got %s", stopper.calledId)
		}
	})

	t.Run("empty helperContainerId", func(t *testing.T) {
		builder := &mockThrottleBuilder{}
		starter := &mockThrottleStarter{}
		stopper := &mockThrottleStopper{}
		executor := &mockThrottleExecutor{}

		throttler := dockersetupcommand.NewDockerContainerThrottler(builder, starter, stopper, executor)
		err := throttler.Unthrottle("eth0", "target-456")
		if err == nil || !strings.Contains(err.Error(), "helper container id is empty") {
			t.Errorf("expected error about empty helper container id, got %v", err)
		}
	})

	t.Run("executor error", func(t *testing.T) {
		builder := &mockThrottleBuilder{returnId: "helper-123"}
		starter := &mockThrottleStarter{}
		stopper := &mockThrottleStopper{}
		executor := &mockThrottleExecutor{returnErr: errors.New("exec error")}

		throttler := dockersetupcommand.NewDockerContainerThrottler(builder, starter, stopper, executor)
		_ = throttler.Throttle("netem loss 5%", "eth0", "target-456")

		err := throttler.Unthrottle("eth0", "target-456")
		if err == nil || !strings.Contains(err.Error(), "failed to execute cleanup command") {
			t.Errorf("expected executor error, got %v", err)
		}
	})

	t.Run("stopper error", func(t *testing.T) {
		builder := &mockThrottleBuilder{returnId: "helper-123"}
		starter := &mockThrottleStarter{}
		stopper := &mockThrottleStopper{returnErr: errors.New("stop error")}
		executor := &mockThrottleExecutor{}

		throttler := dockersetupcommand.NewDockerContainerThrottler(builder, starter, stopper, executor)
		_ = throttler.Throttle("netem loss 5%", "eth0", "target-456")

		err := throttler.Unthrottle("eth0", "target-456")
		if err == nil || !strings.Contains(err.Error(), "failed to stop helper container") {
			t.Errorf("expected stopper error, got %v", err)
		}
	})
}
