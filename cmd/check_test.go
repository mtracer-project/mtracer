package cmd_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/cmd"
	"github.com/mtracer-project/mtracer/configuration"
	"github.com/mtracer-project/mtracer/configuration/openobserve"
	testutils "github.com/mtracer-project/mtracer/testUtils"
)

func TestCheckTestCase_InvalidArgument(t *testing.T) {
	err := cmd.CheckTestCase(nil, []string{"invalid.yaml"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "invalid argument: 'invalid.yaml'. Only .mt.yaml files are allowed"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestCheckTestCase_FolderTrailingSlash(t *testing.T) {
	err := cmd.CheckTestCase(nil, []string{"folder/"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "invalid argument: 'folder/'. Only file names are allowed, not paths"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestCheckTestCase_NonExistentDirectory(t *testing.T) {
	origDir := cmd.Config.Directory
	cmd.Config.Directory = "/nonexistent-path-12345"
	defer func() {
		cmd.Config.Directory = origDir
	}()

	err := cmd.CheckTestCase(nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "error while scanning the directory") {
		t.Fatalf("expected error about scanning directory, got: %v", err)
	}
}

func TestCheckTestCase_ValidAndInvalid(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-check-validity")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	validContent := `
name: "Dice microservice test"
description: "A quick test"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
waitBeforeFetch: 1ms
timeout: 100ms
retryDelay: 10ms
`

	invalidContent := `
name: "Invalid trigger test"
description: "Has invalid trigger type"
trigger:
  type: "unsupported_trigger_type_abc"
  args:
    foo: "bar"
waitBeforeFetch: 1ms
timeout: 100ms
retryDelay: 10ms
`

	testutils.CreateTempYAMLFile(t, tempDir, "valid.mt.yaml", validContent)
	testutils.CreateTempYAMLFile(t, tempDir, "invalid.mt.yaml", invalidContent)

	origConfig := cmd.Config
	cmd.Config = configuration.AppConfig{
		BackendType: "openobserve",
		Directory:   tempDir,
		OpenObserveConfig: &openobserve.OpenObserveConfig{
			BaseURL:    "http://localhost:5080",
			OrgName:    "default",
			StreamName: "default",
			Username:   "admin@example.com",
			Password:   "admin",
		},
	}
	defer func() {
		cmd.Config = origConfig
	}()

	output := testutils.CaptureStdout(t, func() {
		err = cmd.CheckTestCase(nil, nil)
	})

	if err != nil {
		t.Fatalf("expected CheckTestCase to not return error, got: %v", err)
	}

	// Verify valid test outputs "valid"
	if !strings.Contains(output, "valid") {
		t.Errorf("expected output to contain 'valid', got: %q", output)
	}

	// Verify invalid test outputs "invalid"
	if !strings.Contains(output, "invalid") {
		t.Errorf("expected output to contain 'invalid', got: %q", output)
	}

	// Verify that summary table headers are present
	expectedHeaders := []string{"TEST NAME", "STATUS", "DETAILS"}
	for _, h := range expectedHeaders {
		if !strings.Contains(output, h) {
			t.Errorf("expected output to contain table header %q, got: %q", h, output)
		}
	}

	// Verify test status and messages are in the table
	if !strings.Contains(output, "Dice microservice test") {
		t.Errorf("expected output to contain valid test name, got: %q", output)
	}
	if !strings.Contains(output, "Invalid trigger test") {
		t.Errorf("expected output to contain invalid test name, got: %q", output)
	}
	if !strings.Contains(output, "VALID") {
		t.Errorf("expected output to contain 'VALID' status, got: %q", output)
	}
	if !strings.Contains(output, "INVALID") {
		t.Errorf("expected output to contain 'INVALID' status, got: %q", output)
	}
}

func TestCheckTestCase_Quiet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-check-quiet")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	validContent := `
name: "Dice microservice test"
description: "A quick test"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
waitBeforeFetch: 1ms
timeout: 100ms
retryDelay: 10ms
`
	testutils.CreateTempYAMLFile(t, tempDir, "valid.mt.yaml", validContent)

	origConfig := cmd.Config
	cmd.Config = configuration.AppConfig{
		BackendType: "openobserve",
		Directory:   tempDir,
		Quiet:       true, // Enable quiet mode
		OpenObserveConfig: &openobserve.OpenObserveConfig{
			BaseURL:    "http://localhost:5080",
			OrgName:    "default",
			StreamName: "default",
			Username:   "admin@example.com",
			Password:   "admin",
		},
	}
	defer func() {
		cmd.Config = origConfig
	}()

	output := testutils.CaptureStdoutWithLevel(t, slog.LevelWarn, func() {
		err = cmd.CheckTestCase(nil, nil)
	})

	if err != nil {
		t.Fatalf("expected CheckTestCase to not return error, got: %v", err)
	}

	// Verify that table headers and summary details are NOT printed
	if strings.Contains(output, "TEST NAME") || strings.Contains(output, "STATUS") || strings.Contains(output, "DETAILS") {
		t.Errorf("expected summary table to be suppressed, but got it in output: %q", output)
	}
}
