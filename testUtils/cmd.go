package testutils

import (
	"os"
	"path/filepath"
	"testing"
)

// SetupMockExecutable creates a mock executable shell script in a temporary bin directory,
// prepends that directory to the PATH environment variable, and returns the path to a log file
// where invocation arguments are recorded, along with a cleanup function.
//
// cmdName is the name of the command to mock (e.g. "npx" or "docker").
// scriptContent is the shell script body. If empty, a default script is used that logs arguments
// and exits with 0 (or 1 if MTRACER_TEST_FAIL env var is "true").
func SetupMockExecutable(t *testing.T, cmdName string, scriptContent string) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil { // nolint:gosec
		t.Fatalf("failed to create bin dir: %v", err)
	}

	logFile := filepath.Join(tmpDir, "args.log")
	scriptPath := filepath.Join(binDir, cmdName)

	if scriptContent == "" {
		scriptContent = `#!/bin/sh
echo "$@" >> "` + logFile + `"
if [ "$MTRACER_TEST_FAIL" = "true" ]; then
  echo "mock ` + cmdName + ` error output" >&2
  exit 1
else
  echo "mock ` + cmdName + ` success output"
  exit 0
fi
`
	}

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil { // nolint:gosec
		t.Fatalf("failed to write mock script: %v", err)
	}

	oldPath := os.Getenv("PATH")
	newPath := binDir + string(filepath.ListSeparator) + oldPath
	if err := os.Setenv("PATH", newPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}

	cleanup := func() {
		if err := os.Setenv("PATH", oldPath); err != nil {
			t.Fatalf("failed to set PATH: %v", err)
		}
		if err := os.Unsetenv("MTRACER_TEST_FAIL"); err != nil {
			t.Fatalf("failed to unset MTRACER_TEST_FAIL: %v", err)
		}
	}

	return logFile, cleanup
}
