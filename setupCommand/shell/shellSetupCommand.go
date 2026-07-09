package shellsetupcommand

import (
	"context"
	"fmt"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/parser"
)

/*---------------- SetupCommand ----------------*/

type ShellSetupCommand struct {
	cmd        string
	cleanupCmd *ShellCleanupCommand

	baseDir string
	ctx     context.Context
}

func (s *ShellSetupCommand) Execute() error {
	_, err := domain.ExecuteShellCommand(s.cmd, s.baseDir, s.ctx)
	if err != nil {
		return fmt.Errorf("failed to execute setup command '%s': %w", s.cmd, err)
	}
	return nil
}

func NewShellSetupCommand(cmd *parser.SetupCommandDTO, baseDir string, ctx context.Context) (*ShellSetupCommand, error) {
	if cmd == nil {
		return nil, fmt.Errorf("setup command is required")
	}

	if cmd.CleanupCmd == nil {
		return nil, fmt.Errorf("cleanup command is required for shell setup command '%s'", cmd.Cmd)
	}

	return &ShellSetupCommand{
		cmd:        cmd.Cmd,
		baseDir:    baseDir,
		ctx:        ctx,
		cleanupCmd: NewShellCleanupCommand(cmd.CleanupCmd, baseDir, ctx),
	}, nil
}

func (s *ShellSetupCommand) Cleanup() error {
	if s.cleanupCmd != nil {
		return s.cleanupCmd.Cleanup()
	}
	return fmt.Errorf("no cleanup command defined for shell setup command '%s'", s.cmd)
}

/*---------------- CleanupCommand ----------------*/

type ShellCleanupCommand struct {
	cmd string

	baseDir string
	ctx     context.Context
}

func (c *ShellCleanupCommand) Cleanup() error {
	_, err := domain.ExecuteShellCommand(c.cmd, c.baseDir, c.ctx)
	if err != nil {
		return fmt.Errorf("failed to execute cleanup command '%s': %w", c.cmd, err)
	}
	return nil
}

func NewShellCleanupCommand(dto *parser.CleanupCommandDTO, baseDir string, ctx context.Context) *ShellCleanupCommand {
	if dto == nil {
		return nil
	}
	return &ShellCleanupCommand{
		cmd:     dto.Cmd,
		baseDir: baseDir,
		ctx:     ctx,
	}
}
