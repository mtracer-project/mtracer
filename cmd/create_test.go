package cmd_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/cmd"
	testutils "github.com/mtracer-project/mtracer/testUtils"
)

func TestCreateTestCase_NoArgs(t *testing.T) {
	output := testutils.CaptureStdout(t, func() {
		cmd.CreateTestCase(nil, []string{})
	})

	if !strings.Contains(output, "Please provide a name for the test case") {
		t.Errorf("Expected usage warning output, got: %q", output)
	}
}

func TestCreateTestCase_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-cmd-create")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	origDir := cmd.Config.Directory
	origType := cmd.TriggerType

	cmd.Config.Directory = tempDir
	cmd.TriggerType = "http"

	defer func() {
		cmd.Config.Directory = origDir
		cmd.TriggerType = origType
	}()

	testName := "success_test"
	expectedPath := filepath.Join(tempDir, testName+".mt.yaml")

	output := testutils.CaptureStdout(t, func() {
		cmd.CreateTestCase(nil, []string{testName})
	})

	if !strings.Contains(output, "created") || !strings.Contains(output, "successfully") {
		t.Errorf("Expected success output message to contain 'created' and 'successfully', got: %q", output)
	}

	// Verify file was written
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected test file to be created at %s, but it was not", expectedPath)
	}

	content, err := os.ReadFile(expectedPath) // nolint:gosec
	if err != nil {
		t.Fatalf("failed to read created test file: %v", err)
	}

	// Verify trigger example is HTTP (default example)
	if !strings.Contains(string(content), "type: \"http\"") {
		t.Error("Expected created file to have type: \"http\" trigger segment")
	}
}

func TestCreateTestCase_AlreadyExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-cmd-create-exist")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	origDir := cmd.Config.Directory
	cmd.Config.Directory = tempDir
	defer func() {
		cmd.Config.Directory = origDir
	}()

	testName := "duplicate_test"
	existingFilePath := filepath.Join(tempDir, testName+".mt.yaml")

	// Pre-create the file
	err = os.WriteFile(existingFilePath, []byte("existing"), 0o644) // nolint:gosec
	if err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	output := testutils.CaptureStdout(t, func() {
		cmd.CreateTestCase(nil, []string{testName})
	})

	if !strings.Contains(output, "already exists") {
		t.Errorf("Expected conflict/exists message, got: %q", output)
	}
}

func TestCreateTestCase_DifferentTriggerTypes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-cmd-create-triggers")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	origDir := cmd.Config.Directory
	origType := cmd.TriggerType

	cmd.Config.Directory = tempDir
	defer func() {
		cmd.Config.Directory = origDir
		cmd.TriggerType = origType
	}()

	triggerTypes := []string{"traceId", "nats", "jetstream", "grpc", "invalidType"}

	for _, tt := range triggerTypes {
		t.Run(tt, func(t *testing.T) {
			cmd.TriggerType = tt
			testName := "test_" + tt
			expectedPath := filepath.Join(tempDir, testName+".mt.yaml")

			_ = testutils.CaptureStdout(t, func() {
				cmd.CreateTestCase(nil, []string{testName})
			})

			// Verify file creation
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				t.Fatalf("Expected file to exist: %s", expectedPath)
			}

			content, err := os.ReadFile(expectedPath) // nolint:gosec
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if tt == "invalidType" {
				// Should fallback to default trigger (http)
				if !strings.Contains(string(content), "type: \"http\"") {
					t.Error("Expected fallback to default HTTP trigger for invalid type")
				}
			} else {
				// Should write the specific type example
				if !strings.Contains(strings.ToLower(string(content)), "type: \""+strings.ToLower(tt)+"\"") {
					t.Errorf("Expected trigger type %q in generated yaml", tt)
				}
			}
		})
	}
}

func TestCreateTestCase_Quiet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-cmd-create-quiet")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	origDir := cmd.Config.Directory
	origType := cmd.TriggerType
	origConfig := cmd.Config

	cmd.Config.Directory = tempDir
	cmd.Config.Quiet = true // Enable quiet mode
	cmd.TriggerType = "http"

	defer func() {
		cmd.Config.Directory = origDir
		cmd.TriggerType = origType
		cmd.Config = origConfig
	}()

	testName := "quiet_test"
	expectedPath := filepath.Join(tempDir, testName+".mt.yaml")

	output := testutils.CaptureStdoutWithLevel(t, slog.LevelWarn, func() {
		cmd.CreateTestCase(nil, []string{testName})
	})

	// Verify direct fmt.Printf output is suppressed
	if strings.Contains(output, "successfully at") {
		t.Errorf("Expected Printf output to be suppressed when quiet mode is enabled, but got: %q", output)
	}

	// Verify file was still written successfully
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected test file to be created at %s, but it was not", expectedPath)
	}
}
