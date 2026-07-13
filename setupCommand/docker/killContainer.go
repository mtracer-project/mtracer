package dockersetupcommand

import (
	"fmt"
	"log/slog"

	"github.com/mtracer-project/mtracer/parser"

	"github.com/moby/moby/client"
)

type KillContainerCommand struct {
	containerId string // name or id of the container to kill
	killer      ContainerKiller
	starter     ContainerStarter
}

func (s *KillContainerCommand) Execute() error {
	err := s.killer.Kill(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to kill container '%s': %w", s.containerId, err)
	}
	return nil
}

func NewKillContainerCommand(cmd *parser.SetupCommandDTO, killer ContainerKiller, starter ContainerStarter) (*KillContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	containerId, ok := cmd.Args["containerId"].(string)
	if !ok || containerId == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	return &KillContainerCommand{
		containerId: containerId,
		killer:      killer,
		starter:     starter,
	}, nil
}

func (s *KillContainerCommand) Cleanup() error {
	err := s.starter.Start(s.containerId)
	if err != nil {
		return fmt.Errorf("failed to restart container '%s' after kill: %w", s.containerId, err)
	}
	return nil
}

func (e *DockerHandler) Kill(containerId string) error {
	_, err := e.client.ContainerKill(e.ctx, containerId, client.ContainerKillOptions{})
	if err != nil {
		return fmt.Errorf("failed to kill container '%s': %w", containerId, err)
	}

	slog.Info("Container killed successfully", "containerId", containerId)

	return nil
}
