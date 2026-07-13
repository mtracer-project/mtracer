package postexecchecks

import (
	"context"
	"fmt"
	"strings"

	"github.com/mtracer-project/mtracer/parser"
)

type PostExecCheck interface {
	Check() (bool, error)
}

func NewPostExecCheck(dto *parser.PostExecCheckDTO, baseDir string, ctx context.Context) (PostExecCheck, error) {
	switch strings.ToLower(dto.Type) {
	case "sql":
		return NewSQLPostExecCheck(dto, ctx)
	case "shell":
		return NewShellPostExecCheck(dto, baseDir, ctx)
	default:
		return nil, fmt.Errorf("unsupported post exec check type: %s", dto.Type)
	}
}

func NewPostExecChecks(dtos []*parser.PostExecCheckDTO, baseDir string, ctx context.Context) ([]PostExecCheck, error) {
	var postExecChecks []PostExecCheck
	for _, dto := range dtos {
		check, err := NewPostExecCheck(dto, baseDir, ctx)
		if err != nil {
			return nil, fmt.Errorf("error creating post exec check: %w", err)
		}
		postExecChecks = append(postExecChecks, check)
	}
	return postExecChecks, nil
}
