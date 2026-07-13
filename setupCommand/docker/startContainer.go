package dockersetupcommand

import (
	"fmt"
	"log/slog"

	"github.com/mtracer-project/mtracer/parser"

	"github.com/moby/moby/client"
)

type StartContainerCommand struct {
	containerId string // name or id of the container to start
	starter     ContainerStarter
	stopper     ContainerStopper
}

func (s *StartContainerCommand) Execute() error {
	err := s.starter.Start(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to start container '%s': %w", s.containerId, err)
	}
	return nil
}

func NewStartContainerCommand(cmd *parser.SetupCommandDTO, starter ContainerStarter, stopper ContainerStopper) (*StartContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	containerId, ok := cmd.Args["containerId"].(string)
	if !ok || containerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	return &StartContainerCommand{
		containerId: containerId,
		starter:     starter,
		stopper:     stopper,
	}, nil
}

func (s *StartContainerCommand) Cleanup() error {
	err := s.stopper.Stop(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to stop container '%s' during cleanup: %w", s.containerId, err)
	}
	return nil
}

func (e *DockerHandler) Start(containerId string) error {
	_, err := e.client.ContainerStart(e.ctx, containerId, client.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container '%s': %w", containerId, err)
	}

	slog.Info("Container started successfully", "containerId", containerId)

	return nil
}
