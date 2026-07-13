package setupcommand_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	setupcommand "github.com/mtracer-project/mtracer/setupCommand"
)

func TestNewSetupCommand(t *testing.T) {
	t.Run("valid shell command type", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Type: "shell",
			Cmd:  "echo 'test'",
			CleanupCmd: &parser.CleanupCommandDTO{
				Cmd: "echo 'cleanup'",
			},
		}

		cmd, err := setupcommand.NewSetupCommand(dto, nil, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("valid docker command type", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Type: "docker",
			Cmd:  "killcontainer",
			Args: map[string]any{
				"containerId": "test-container",
			},
		}

		cmd, err := setupcommand.NewSetupCommand(dto, nil, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cmd == nil {
			t.Fatal("expected command to be non-nil")
		}
	})

	t.Run("unsupported command type", func(t *testing.T) {
		dto := &parser.SetupCommandDTO{
			Type: "invalid-type",
			Cmd:  "echo 'test'",
		}

		_, err := setupcommand.NewSetupCommand(dto, nil, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "unsupported setup command type") {
			t.Errorf("expected error about unsupported type, got: %v", err)
		}
	})
}

func TestNewSetupCommands(t *testing.T) {
	t.Run("success slice of commands", func(t *testing.T) {
		dtos := []*parser.SetupCommandDTO{
			{
				Type: "shell",
				Cmd:  "echo 'cmd 1'",
				CleanupCmd: &parser.CleanupCommandDTO{
					Cmd: "echo 'clean 1'",
				},
			},
			{
				Type: "shell",
				Cmd:  "echo 'cmd 2'",
				CleanupCmd: &parser.CleanupCommandDTO{
					Cmd: "echo 'clean 2'",
				},
			},
		}

		cmds, err := setupcommand.NewSetupCommands(dtos, nil, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(cmds) != 2 {
			t.Errorf("expected 2 commands, got %d", len(cmds))
		}
	})

	t.Run("success mixed shell and docker commands", func(t *testing.T) {
		dtos := []*parser.SetupCommandDTO{
			{
				Type: "shell",
				Cmd:  "echo 'cmd 1'",
				CleanupCmd: &parser.CleanupCommandDTO{
					Cmd: "echo 'clean 1'",
				},
			},
			{
				Type: "docker",
				Cmd:  "stopcontainer",
				Args: map[string]any{
					"containerId": "test-container-2",
				},
			},
		}

		cmds, err := setupcommand.NewSetupCommands(dtos, nil, "", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(cmds) != 2 {
			t.Errorf("expected 2 commands, got %d", len(cmds))
		}
	})

	t.Run("failure on one of the commands", func(t *testing.T) {
		dtos := []*parser.SetupCommandDTO{
			{
				Type: "shell",
				Cmd:  "echo 'cmd 1'",
				CleanupCmd: &parser.CleanupCommandDTO{
					Cmd: "echo 'clean 1'",
				},
			},
			{
				Type: "invalid-type",
				Cmd:  "echo 'cmd 2'",
			},
		}

		_, err := setupcommand.NewSetupCommands(dtos, nil, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "error creating setup command") {
			t.Errorf("expected error about setup command creation, got: %v", err)
		}
	})
}
