package dockersetupcommand

import (
	"fmt"
	"log/slog"

	"github.com/mtrace-project/mtrace/parser"

	"github.com/moby/moby/client"
)

type StopContainerCommand struct {
	containerId string // name or id of the container to stop
	starter     ContainerStarter
	stopper     ContainerStopper
}

func (s *StopContainerCommand) Execute() error {
	err := s.stopper.Stop(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to stop container '%s': %w", s.containerId, err)
	}
	return nil
}

func NewStopContainerCommand(cmd *parser.SetupCommandDTO, stopper ContainerStopper, starter ContainerStarter) (*StopContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	containerId, ok := cmd.Args["containerId"].(string)
	if !ok || containerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	return &StopContainerCommand{
		containerId: containerId,
		stopper:     stopper,
		starter:     starter,
	}, nil
}

func (s *StopContainerCommand) Cleanup() error {
	err := s.starter.Start(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to start container '%s' after stop: %w", s.containerId, err)
	}
	return nil
}

func (e *DockerHandler) Stop(containerId string) error {
	_, err := e.client.ContainerStop(e.ctx, containerId, client.ContainerStopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container '%s' after start: %w", containerId, err)
	}

	slog.Info("Container stopped successfully", "containerId", containerId)

	return nil
}
