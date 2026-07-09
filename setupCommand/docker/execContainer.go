package dockersetupcommand

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mtrace-project/mtrace/parser"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

/*---------------- ExecContainerCommand ----------------*/

type ExecContainerCommand struct {
	containerId string // name or id of the container to exec into
	cmd         string // command and args to execute
	executor    ContainerCommandExecutor
	cleanupCmd  *ExecContainerCleanupCommand
}

func (s *ExecContainerCommand) Execute() error {
	if err := s.executor.Execute(s.containerId, s.cmd); err != nil {
		return fmt.Errorf("failed to execute command in container '%s': %w", s.containerId, err)
	}
	return nil
}

func NewExecContainerCommand(cmd *parser.SetupCommandDTO, executor ContainerCommandExecutor) (*ExecContainerCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	if cmd.CleanupCmd == nil {
		return nil, fmt.Errorf("cleanup command is required for exec container setup command '%s'", cmd.Cmd)
	}

	containerId, ok := cmd.Args["containerId"].(string)
	if !ok || strings.TrimSpace(containerId) == "" {
		return nil, fmt.Errorf("containerId argument is required and must be a non-empty string for exec container setup command '%s'", cmd.Type)
	}

	execCmd, ok := cmd.Args["cmd"].(string)
	if !ok || strings.TrimSpace(execCmd) == "" {
		return nil, fmt.Errorf("cmd argument is required and must be a non-empty string for exec container setup command '%s'", cmd.Type)
	}

	cleanupCmd, err := NewExecContainerCleanupCommand(cmd.CleanupCmd, containerId, executor)
	if err != nil {
		return nil, fmt.Errorf("error creating cleanup command for exec container setup command '%s': %w", cmd.Cmd, err)
	}

	return &ExecContainerCommand{
		containerId: containerId,
		cmd:         execCmd,
		executor:    executor,
		cleanupCmd:  cleanupCmd,
	}, nil
}

func (s *ExecContainerCommand) Cleanup() error {
	if s.cleanupCmd != nil {
		return s.cleanupCmd.Cleanup()
	}
	return fmt.Errorf("no cleanup command defined for exec container setup command")
}

/*---------------- ExecContainerCleanupCommand ----------------*/

type ExecContainerCleanupCommand struct {
	containerId string
	cmd         string
	executor    ContainerCommandExecutor
}

func (c *ExecContainerCleanupCommand) Cleanup() error {
	if err := c.executor.Execute(c.containerId, c.cmd); err != nil {
		return fmt.Errorf("failed to execute cleanup command in container '%s': %w", c.containerId, err)
	}

	return nil
}

func NewExecContainerCleanupCommand(cleanupCmd *parser.CleanupCommandDTO, containerId string, executor ContainerCommandExecutor) (*ExecContainerCleanupCommand, error) {
	if cleanupCmd == nil {
		return nil, fmt.Errorf("cleanup command DTO is required")
	}

	if strings.TrimSpace(cleanupCmd.Cmd) == "" {
		return nil, fmt.Errorf("cleanup command string is required and cannot be empty for exec container cleanup command")
	}

	return &ExecContainerCleanupCommand{
		containerId: containerId,
		executor:    executor,
		cmd:         cleanupCmd.Cmd,
	}, nil
}

// ALERT: this Execute method is used to exec a command in a container, meanwhile the Execute method of the ExecContainerCommand is used to execute the command defined in the setup command
func (e *DockerHandler) Execute(containerId string, cmd string) error {
	execConfig := client.ExecCreateOptions{
		Cmd:          strings.Split(cmd, " "),
		AttachStdout: true,
		AttachStderr: true,
	}

	execResult, err := e.client.ExecCreate(e.ctx, containerId, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec in container '%s': %w", containerId, err)
	}

	if slog.Default().Enabled(e.ctx, slog.LevelInfo) {
		attachResult, err := e.client.ExecAttach(e.ctx, execResult.ID, client.ExecAttachOptions{})
		if err != nil {
			return fmt.Errorf("failed to attach to exec in container '%s': %w", containerId, err)
		}
		defer attachResult.Close()

		if _, err := e.client.ExecStart(e.ctx, execResult.ID, client.ExecStartOptions{Detach: false}); err != nil {
			return fmt.Errorf("failed to start exec in container '%s': %w", containerId, err)
		}

		var outBuf, errBuf bytes.Buffer
		if _, err := stdcopy.StdCopy(&outBuf, &errBuf, attachResult.Reader); err != nil {
			return fmt.Errorf("error reading exec stream: %w", err)
		}

		slog.Info(
			"Executed command in container",
			"containerId", containerId,
			"cmd", cmd,
			"stdout", strings.TrimSpace(outBuf.String()),
			"stderr", strings.TrimSpace(errBuf.String()),
		)
	} else {
		if _, err := e.client.ExecStart(e.ctx, execResult.ID, client.ExecStartOptions{Detach: true}); err != nil {
			return fmt.Errorf("failed to start exec in container '%s': %w", containerId, err)
		}
	}

	return nil
}
