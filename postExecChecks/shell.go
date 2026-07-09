package postexecchecks

import (
	"context"
	"fmt"

	"github.com/mtrace-project/mtrace/domain"
	"github.com/mtrace-project/mtrace/parser"
)

const (
	DEFAULT_EXPECTED_EXIT_CODE = 0
)

type ShellPostExecCheck struct {
	name             string
	cmd              string
	expectedExitCode int

	baseDir string
	ctx     context.Context
}

func (s *ShellPostExecCheck) Check() (bool, error) {
	exitCode, err := domain.ExecuteShellCommand(s.cmd, s.baseDir, s.ctx)
	if err != nil {
		return false, fmt.Errorf("check failed for command '%s': %w", s.cmd, err)
	}
	return exitCode == s.expectedExitCode, nil
}

func NewShellPostExecCheck(dto *parser.PostExecCheckDTO, baseDir string, ctx context.Context) (*ShellPostExecCheck, error) {
	if dto.Args == nil {
		return nil, fmt.Errorf("args are required for shell post exec check")
	}

	cmd, ok := dto.Args["cmd"].(string)
	if !ok {
		return nil, fmt.Errorf("cmd is required for shell post exec check")
	}

	expectedExitCode, ok := dto.Args["expectedExitCode"].(int)
	if !ok {
		expectedExitCode = DEFAULT_EXPECTED_EXIT_CODE
	}

	return &ShellPostExecCheck{
		name:             dto.Name,
		cmd:              cmd,
		expectedExitCode: expectedExitCode,
		baseDir:          baseDir,
		ctx:              ctx,
	}, nil
}
