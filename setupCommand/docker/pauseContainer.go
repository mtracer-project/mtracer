package dockersetupcommand

import (
	"fmt"
	"log/slog"

	"github.com/mtrace-project/mtrace/parser"

	"github.com/moby/moby/client"
)

type PauseContainerCommand struct {
	containerId string // name or id of the container to pause
	pauser      ContainerPauser
	unpauser    ContainerUnpauser
}

func (s *PauseContainerCommand) Execute() error {
	err := s.pauser.Pause(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to pause container '%s': %w", s.containerId, err)
	}
	return nil
}

func NewPauseContainerCommand(cmd *parser.SetupCommandDTO, pauser ContainerPauser, unpauser ContainerUnpauser) (*PauseContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	containerId, ok := cmd.Args["containerId"].(string)
	if !ok || containerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	return &PauseContainerCommand{
		containerId: containerId,
		pauser:      pauser,
		unpauser:    unpauser,
	}, nil
}

func (s *PauseContainerCommand) Cleanup() error {
	err := s.unpauser.Unpause(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to unpause container '%s' after pause: %w", s.containerId, err)
	}
	return nil
}

func (e *DockerHandler) Pause(containerId string) error {
	_, err := e.client.ContainerPause(e.ctx, containerId, client.ContainerPauseOptions{})
	if err != nil {
		return fmt.Errorf("failed to pause container '%s': %w", containerId, err)
	}

	slog.Info("Container paused successfully", "containerId", containerId)

	return nil
}
