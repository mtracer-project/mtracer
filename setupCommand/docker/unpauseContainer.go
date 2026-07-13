package dockersetupcommand

import (
	"fmt"
	"log/slog"

	"github.com/mtracer-project/mtracer/parser"

	"github.com/moby/moby/client"
)

type UnpauseContainerCommand struct {
	containerId string // name or id of the container to unpause
	unpauser    ContainerUnpauser
	pauser      ContainerPauser
}

func (s *UnpauseContainerCommand) Execute() error {
	err := s.unpauser.Unpause(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to unpause container '%s': %w", s.containerId, err)
	}
	return nil
}

func NewUnpauseContainerCommand(cmd *parser.SetupCommandDTO, unpauser ContainerUnpauser, pauser ContainerPauser) (*UnpauseContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	containerId, ok := cmd.Args["containerId"].(string)
	if !ok || containerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	return &UnpauseContainerCommand{
		containerId: containerId,
		unpauser:    unpauser,
		pauser:      pauser,
	}, nil
}

func (s *UnpauseContainerCommand) Cleanup() error {
	err := s.pauser.Pause(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to pause container '%s' after unpause: %w", s.containerId, err)
	}
	return nil
}

func (e *DockerHandler) Unpause(containerId string) error {
	_, err := e.client.ContainerUnpause(e.ctx, containerId, client.ContainerUnpauseOptions{})
	if err != nil {
		return fmt.Errorf("failed to unpause container '%s': %w", containerId, err)
	}

	slog.Info("Container unpaused successfully", "containerId", containerId)

	return nil
}
