package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	testutils "github.com/mtrace-project/mtrace/testUtils"
)

func TestMangenCmd(t *testing.T) {
	// Change working directory to a temp dir so that "man" is created there
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("unexpected error getting wd: %v", err)
	}

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("unexpected error changing dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
	}()

	output := testutils.CaptureStdout(t, func() {
		gendocsCmd.Run(gendocsCmd, nil)
	})

	if !strings.Contains(output, "Man pages generated with success at") {
		t.Errorf("expected success message, got: %v", output)
	}

	expectedPath := filepath.Join(tempDir, "man", "mtrace.1")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected man page to be generated at %s", expectedPath)
	}
}
