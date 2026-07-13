package trigger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	testutils "github.com/mtracer-project/mtracer/testUtils"
)

func TestPlaywrightTrigger_Init(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{}

	t.Run("missing filePath", func(t *testing.T) {
		dto := &parser.TriggerDTO{
			Type: "playwright",
			Args: map[string]any{},
		}
		var tr PlaywrightTrigger
		err := tr.Init(dto, mockIDGen, "", context.Background())
		if err == nil || !strings.Contains(err.Error(), "filePath is required") {
			t.Errorf("expected error about filePath is required, got: %v", err)
		}
	})

	t.Run("valid config with defaults", func(t *testing.T) {
		dto := &parser.TriggerDTO{
			Type: "playwright",
			Args: map[string]any{
				"filePath": "my-test.spec.ts",
			},
		}
		var tr PlaywrightTrigger
		err := tr.Init(dto, mockIDGen, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.filePath != "my-test.spec.ts" {
			t.Errorf("expected filePath to be 'my-test.spec.ts', got %q", tr.filePath)
		}
		if tr.playwrightPath != "playwright" {
			t.Errorf("expected playwrightPath to be 'playwright', got %q", tr.playwrightPath)
		}
		if tr.traceUrlPattern != "**" {
			t.Errorf("expected traceUrlPattern to be '**', got %q", tr.traceUrlPattern)
		}
		if len(tr.projects) != 0 {
			t.Errorf("expected empty projects, got %v", tr.projects)
		}
	})

	t.Run("custom args", func(t *testing.T) {
		dto := &parser.TriggerDTO{
			Type: "playwright",
			Args: map[string]any{
				"filePath":        "test.spec.ts",
				"playwrightPath":  "custom-pw-dir",
				"traceUrlPattern": "**/api/*",
				"projects":        []any{"chromium", "firefox", ""},
			},
		}
		var tr PlaywrightTrigger
		err := tr.Init(dto, mockIDGen, "/base", context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.playwrightPath != "custom-pw-dir" {
			t.Errorf("expected playwrightPath to be 'custom-pw-dir', got %q", tr.playwrightPath)
		}
		if tr.traceUrlPattern != "**/api/*" {
			t.Errorf("expected traceUrlPattern to be '**/api/*', got %q", tr.traceUrlPattern)
		}
		expectedProjects := []string{"chromium", "firefox"}
		if len(tr.projects) != 2 || tr.projects[0] != "chromium" || tr.projects[1] != "firefox" {
			t.Errorf("expected projects %v, got %v", expectedProjects, tr.projects)
		}
	})
}

func TestPlaywrightTraceIdServer(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: "12345678901234567890123456789012",
		SpanID:  "1234567890123456",
	}

	server := &playwrightTraceIdServer{
		serverAddress:   "localhost",
		port:            0,
		idGenerator:     mockIDGen,
		ctx:             context.Background(),
		traceUrlPattern: "**/api/*",
	}

	traceIdChan := make(chan TraceId, 1)
	srv, err := server.start(traceIdChan)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Shutdown(context.Background()) // nolint:errcheck

	// Make HTTP call to it
	resp, err := http.Get(fmt.Sprintf("http://%s%s", srv.Addr, PLAYWRIGHT_SERVER_ENDPOINT)) // nolint:noctx
	if err != nil {
		t.Fatalf("HTTP GET request failed: %v", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var resObj traceResponse
	err = json.Unmarshal(body, &resObj)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}

	expectedTraceparent := "00-12345678901234567890123456789012-1234567890123456-01"
	if resObj.Traceparent != expectedTraceparent {
		t.Errorf("expected traceparent %q, got %q", expectedTraceparent, resObj.Traceparent)
	}

	expectedTraceUrlPattern := "**/api/*"
	if resObj.TraceUrlPattern != expectedTraceUrlPattern {
		t.Errorf("expected traceUrlPattern %q, got %q", expectedTraceUrlPattern, resObj.TraceUrlPattern)
	}

	var traceId TraceId
	select {
	case traceId = <-traceIdChan:
	default:
	}

	if traceId != "12345678901234567890123456789012" {
		t.Errorf("expected traceId variable to be updated to %q, got %q", "12345678901234567890123456789012", traceId)
	}
}

func TestPlaywrightTrigger_Trigger_Success(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: "abcdefabcdefabcdefabcdefabcdef12",
		SpanID:  "abcdefabcdefabcdef",
	}

	// We create a mock npx script that curls/wgets the MTRACER_PLAYWRIGHT_SERVER_URL
	scriptContent := `#!/bin/sh
if [ -n "$MTRACER_PLAYWRIGHT_SERVER_URL" ]; then
  if command -v curl >/dev/null 2>&1; then
    curl -s "$MTRACER_PLAYWRIGHT_SERVER_URL" > /dev/null
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O /dev/null "$MTRACER_PLAYWRIGHT_SERVER_URL"
  else
    echo "Neither curl nor wget found" >&2
    exit 1
  fi
fi
exit 0
`
	_, cleanup := testutils.SetupMockExecutable(t, "npx", scriptContent)
	defer cleanup()

	dto := &parser.TriggerDTO{
		Type: "playwright",
		Args: map[string]any{
			"filePath": "test.spec.ts",
		},
	}

	var trigger PlaywrightTrigger
	baseDir := t.TempDir()
	err := trigger.Init(dto, mockIDGen, baseDir, context.Background())
	if err != nil {
		t.Fatalf("unexpected error on Init: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(baseDir, trigger.playwrightPath), 0o755); err != nil { // nolint:gosec
		t.Fatalf("failed to create playwright path: %v", err)
	}

	// Run Trigger
	traceId, err := trigger.Trigger()
	if err != nil {
		t.Fatalf("unexpected error on Trigger: %v", err)
	}

	if traceId != "abcdefabcdefabcdefabcdefabcdef12" {
		t.Errorf("expected traceId %q, got %q", "abcdefabcdefabcdefabcdefabcdef12", traceId)
	}
}

func TestPlaywrightTrigger_Trigger_CommandFailure(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: "abcdefabcdefabcdefabcdefabcdef12",
		SpanID:  "abcdefabcdefabcdef",
	}

	scriptContent := `#!/bin/sh
exit 1
`
	_, cleanup := testutils.SetupMockExecutable(t, "npx", scriptContent)
	defer cleanup()

	dto := &parser.TriggerDTO{
		Type: "playwright",
		Args: map[string]any{
			"filePath": "test.spec.ts",
		},
	}

	var trigger PlaywrightTrigger
	baseDir := t.TempDir()
	err := trigger.Init(dto, mockIDGen, baseDir, context.Background())
	if err != nil {
		t.Fatalf("unexpected error on Init: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(baseDir, trigger.playwrightPath), 0o755); err != nil { // nolint:gosec
		t.Fatalf("failed to create playwright path: %v", err)
	}

	_, err = trigger.Trigger()
	if err == nil || !strings.Contains(err.Error(), "failed to execute Playwright test") {
		t.Errorf("expected 'failed to execute Playwright test' error, got: %v", err)
	}
}

func TestPlaywrightTrigger_Trigger_MissingTraceId(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: "abcdefabcdefabcdefabcdefabcdef12",
		SpanID:  "abcdefabcdefabcdef",
	}

	scriptContent := `#!/bin/sh
exit 0
`
	_, cleanup := testutils.SetupMockExecutable(t, "npx", scriptContent)
	defer cleanup()

	dto := &parser.TriggerDTO{
		Type: "playwright",
		Args: map[string]any{
			"filePath": "test.spec.ts",
		},
	}

	var trigger PlaywrightTrigger
	baseDir := t.TempDir()
	err := trigger.Init(dto, mockIDGen, baseDir, context.Background())
	if err != nil {
		t.Fatalf("unexpected error on Init: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(baseDir, trigger.playwrightPath), 0o755); err != nil { // nolint:gosec
		t.Fatalf("failed to create playwright path: %v", err)
	}

	_, err = trigger.Trigger()
	if err == nil || !strings.Contains(err.Error(), "traceId was not set by the Playwright test") {
		t.Errorf("expected 'traceId was not set' error, got: %v", err)
	}
}

type mockTraceIdServer struct {
	err error
}

func (m *mockTraceIdServer) start(traceIdChan chan<- TraceId) (*http.Server, error) {
	return nil, m.err
}

func TestPlaywrightTrigger_Trigger_ServerStartFailure(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{}
	dto := &parser.TriggerDTO{
		Type: "playwright",
		Args: map[string]any{
			"filePath": "test.spec.ts",
		},
	}

	var trigger PlaywrightTrigger
	err := trigger.Init(dto, mockIDGen, t.TempDir(), context.Background())
	if err != nil {
		t.Fatalf("unexpected error on Init: %v", err)
	}

	// Inject failing server mock
	trigger.server = &mockTraceIdServer{
		err: errors.New("bind port failure"),
	}

	_, err = trigger.Trigger()
	if err == nil || !strings.Contains(err.Error(), "failed to start traceId server") {
		t.Errorf("expected 'failed to start traceId server' error, got: %v", err)
	}
}

func (t *PlaywrightTrigger) GetExample() string {
	return t.Example()
}

func TestPlaywrightTrigger_Example(t *testing.T) {
	var tr PlaywrightTrigger
	ex := tr.Example()
	if !strings.Contains(ex, "playwright") {
		t.Errorf("expected Example to contain 'playwright', got %q", ex)
	}
}
