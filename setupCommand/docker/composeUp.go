package dockersetupcommand

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"

	"github.com/mtrace-project/mtrace/parser"
)

type ComposeUpCommand struct {
	projectName string // name of the Docker Compose project
	composePath string // path to the Docker Compose file
	ctx         context.Context
}

func getComposeUpCommand(composePath string, projectName string, ctx context.Context) *exec.Cmd {
	cmds := []string{"docker", "compose", "-f", composePath}
	if projectName != "" {
		cmds = append(cmds, "-p", projectName)
	}
	cmds = append(cmds, "up", "-d", "--wait")

	return exec.CommandContext(ctx, cmds[0], cmds[1:]...)
}

func getComposeDownCommand(composePath string, projectName string, ctx context.Context) *exec.Cmd {
	cmds := []string{"docker", "compose", "-f", composePath}
	if projectName != "" {
		cmds = append(cmds, "-p", projectName)
	}
	cmds = append(cmds, "down", "-v", "--remove-orphans")
	return exec.CommandContext(ctx, cmds[0], cmds[1:]...)
}

func (s *ComposeUpCommand) Execute() error {
	cmd := getComposeUpCommand(s.composePath, s.projectName, s.ctx)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %s: %w", string(output), err)
	}

	slog.Info("Docker Compose started successfully", "output", string(output))

	return nil
}

func NewComposeUpCommand(cmd *parser.SetupCommandDTO, baseDir string, ctx context.Context) (*ComposeUpCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	projectName, ok := cmd.Args["projectName"].(string)
	if !ok {
		projectName = ""
	}

	composePath, ok := cmd.Args["composePath"].(string)
	if !ok || composePath == "" {
		return nil, fmt.Errorf("composePath argument is required and must be a non-empty string for setup command '%s'", cmd.Cmd)
	}

	composePath = filepath.Join(baseDir, composePath)

	return &ComposeUpCommand{
		projectName: projectName,
		composePath: composePath,
		ctx:         ctx,
	}, nil
}

func (s *ComposeUpCommand) Cleanup() error {
	cmd := getComposeDownCommand(s.composePath, s.projectName, s.ctx)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose down failed: %s: %w", string(output), err)
	}
	slog.Info("Docker Compose stopped successfully", "output", string(output))
	return nil
}
