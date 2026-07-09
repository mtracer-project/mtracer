package cmd_test

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/cmd"
	"github.com/mtrace-project/mtrace/configuration"
	"github.com/mtrace-project/mtrace/configuration/jaeger"
	"github.com/mtrace-project/mtrace/configuration/openobserve"
	testutils "github.com/mtrace-project/mtrace/testUtils"
)

func TestRunTests_InvalidArgument(t *testing.T) {
	err := cmd.RunTests(nil, []string{"invalid.yaml"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "invalid argument: 'invalid.yaml'. Only .mt.yaml files are allowed"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestRunTests_InvalidExportFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-test-invalid-export")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	testContent := `
name: "Dummy test"
description: "A quick test"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
`
	testutils.CreateTempYAMLFile(t, tempDir, "test.mt.yaml", testContent)

	origFormats := cmd.ExportFormats
	cmd.ExportFormats = []string{"invalid_format"}
	origDir := cmd.Config.Directory
	cmd.Config.Directory = tempDir

	// Create dummy OpenObserve backend so it doesn't fail on NewTraceAdapterFromConfig
	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		return nil, 404
	})
	origConfig := cmd.Config
	cmd.Config.BackendType = "openobserve"
	cmd.Config.OpenObserveConfig = &openobserve.OpenObserveConfig{BaseURL: mockServer.URL}

	defer func() {
		cmd.ExportFormats = origFormats
		cmd.Config = origConfig
		cmd.Config.Directory = origDir
	}()

	err = cmd.RunTests(nil, []string{})
	if err == nil {
		t.Fatal("expected error for invalid export format, got nil")
	}
	if !strings.Contains(err.Error(), "error creating exporters") {
		t.Fatalf("expected unsupported export format error, got %v", err)
	}
}

func TestRunTests_FolderTrailingSlash(t *testing.T) {
	err := cmd.RunTests(nil, []string{"folder/"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "invalid argument: 'folder/'. Only file names are allowed, not paths"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestRunTests_NonExistentDirectory(t *testing.T) {
	origDir := cmd.Config.Directory
	cmd.Config.Directory = "/nonexistent-path-12345"
	defer func() {
		cmd.Config.Directory = origDir
	}()

	err := cmd.RunTests(nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "error while scanning the directory") {
		t.Fatalf("expected scanning error, got %v", err)
	}
}

func TestRunTests_NoFilesFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-test-empty")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	origDir := cmd.Config.Directory
	cmd.Config.Directory = tempDir
	defer func() {
		cmd.Config.Directory = origDir
	}()

	err = cmd.RunTests(nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no .mt.yaml files found to execute") {
		t.Fatalf("expected 'no .mt.yaml files' error, got %v", err)
	}
}

func TestRunTests_ParseError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-test-parse-err")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	// write malformed yaml file
	testutils.CreateTempYAMLFile(t, tempDir, "bad.mt.yaml", `
name: "Malformed yaml
trigger:
`)

	origDir := cmd.Config.Directory
	cmd.Config.Directory = tempDir
	defer func() {
		cmd.Config.Directory = origDir
	}()

	err = cmd.RunTests(nil, nil)
	if err == nil {
		t.Fatal("expected parsing error, got nil")
	}
	if !strings.Contains(err.Error(), "error parsing test file") {
		t.Fatalf("expected parsing error, got %v", err)
	}
}

func TestRunTests_E2E_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-test-e2e-success")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	// Set up mock HTTP server
	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		// Verify query is selecting correct traceId
		if !strings.Contains(sqlQuery, "trace_id = '1234567890abcdef1234567890abcdef'") {
			return nil, http.StatusBadRequest
		}

		hits := []map[string]any{
			{
				"span_id":                  "span1",
				"reference_parent_span_id": "",
				"service_name":             "test-service",
				"operation_name":           "test-op",
				"span_kind":                "server",
				"span_status":              "ok",
				"start_time":               float64(1717495000000000000),
				"end_time":                 float64(1717495005000000000),
				"duration":                 float64(5000000000),
			},
		}
		return hits, http.StatusOK
	})

	// Setup YAML test definition
	testContent := `
name: "Dice microservice test"
description: "A quick test"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
waitBeforeFetch: 1ms
expectedTraces:
  - spans:
      - serviceName: "test-service"
        operationName: "test-op"
        spanKind: "server"
        spanStatus: "ok"
lastSpan:
  serviceName: "test-service"
  operationName: "test-op"
  spanKind: "server"
  spanStatus: "ok"
timeout: 100ms
retryDelay: 10ms
`

	testutils.CreateTempYAMLFile(t, tempDir, "test.mt.yaml", testContent)

	// Save/Restore globals
	origConfig := cmd.Config

	cmd.Config = configuration.AppConfig{
		BackendType: "openobserve",
		Directory:   tempDir,
		Verbose:     false,
		OpenObserveConfig: &openobserve.OpenObserveConfig{
			BaseURL:    mockServer.URL,
			OrgName:    "default",
			StreamName: "default",
			Username:   "admin@example.com",
			Password:   "admin",
		},
	}

	defer func() {
		cmd.Config = origConfig
	}()

	var runErr error
	output := testutils.CaptureStdout(t, func() {
		runErr = cmd.RunTests(nil, nil)
	})

	if runErr != nil {
		t.Fatalf("expected run to succeed, got error: %v", runErr)
	}

	if !strings.Contains(output, "msg=PASSED") || !strings.Contains(output, "testName=\"Dice microservice test\"") {
		t.Fatalf("expected output to contain passed message, got: %q", output)
	}
}

func TestRunTests_E2E_Failure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-test-e2e-failure")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	// Mock server returns NOT found or error
	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		return nil, http.StatusNotFound
	})

	testContent := `
name: "Failing test"
description: "A quick test"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
waitBeforeFetch: 1ms
timeout: 50ms
retryDelay: 10ms
`

	testutils.CreateTempYAMLFile(t, tempDir, "test_fail.mt.yaml", testContent)

	// Save/Restore globals
	origConfig := cmd.Config

	cmd.Config = configuration.AppConfig{
		BackendType: "openobserve",
		Directory:   tempDir,
		Verbose:     false,
		OpenObserveConfig: &openobserve.OpenObserveConfig{
			BaseURL:    mockServer.URL,
			OrgName:    "default",
			StreamName: "default",
			Username:   "admin@example.com",
			Password:   "admin",
		},
	}

	defer func() {
		cmd.Config = origConfig
	}()

	var runErr error
	output := testutils.CaptureStdout(t, func() {
		runErr = cmd.RunTests(nil, nil)
	})

	if runErr != nil {
		t.Fatalf("expected run to succeed (but print failure), got error: %v", runErr)
	}

	if !strings.Contains(output, "msg=FAILED") || !strings.Contains(output, "testName=\"Failing test\"") {
		t.Fatalf("expected output to contain failure message, got: %q", output)
	}
}

func TestRunTests_Jaeger_E2E_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-test-jaeger-e2e-success")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	// Set up mock HTTP server
	mockServer := testutils.StartMockJaegerServer(t, func(traceID string) (any, int) {
		if traceID != "1234567890abcdef1234567890abcdef" {
			return nil, http.StatusBadRequest
		}

		mockSpans := []jaeger.JaegerSpanDTO{
			{
				TraceId:       traceID,
				SpanId:        "span1",
				OperationName: "test-op",
				References:    nil,
				StartTimeUs:   1717495000000000,
				DurationUs:    5000000,
				Tags: []jaeger.JaegerTag{
					{Key: "span.kind", Type: "string", Value: "server"},
					{Key: "otel.status_code", Type: "string", Value: "OK"},
				},
				ProcessID: "p1",
			},
		}

		mockProcesses := map[string]jaeger.JaegerProcess{
			"p1": {ServiceName: "test-service"},
		}

		mockTrace := []jaeger.JaegerTraceDTO{
			{
				TraceId:   traceID,
				Spans:     mockSpans,
				Processes: mockProcesses,
			},
		}

		return mockTrace, http.StatusOK
	})

	// Setup YAML test definition
	testContent := `
name: "Dice microservice jaeger test"
description: "A quick test for jaeger"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
waitBeforeFetch: 1ms
expectedTraces:
  - spans:
      - serviceName: "test-service"
        operationName: "test-op"
        spanKind: "server"
        spanStatus: "ok"
lastSpan:
  serviceName: "test-service"
  operationName: "test-op"
  spanKind: "server"
  spanStatus: "ok"
timeout: 100ms
retryDelay: 10ms
`

	testutils.CreateTempYAMLFile(t, tempDir, "test.mt.yaml", testContent)

	// Save/Restore globals
	origConfig := cmd.Config

	cmd.Config = configuration.AppConfig{
		BackendType: "jaeger",
		Directory:   tempDir,
		Verbose:     false,
		JaegerConfig: &jaeger.JaegerConfig{
			BaseURL: mockServer.URL,
		},
	}

	defer func() {
		cmd.Config = origConfig
	}()

	var runErr error
	output := testutils.CaptureStdout(t, func() {
		runErr = cmd.RunTests(nil, nil)
	})

	if runErr != nil {
		t.Fatalf("expected run to succeed, got error: %v", runErr)
	}

	if !strings.Contains(output, "msg=PASSED") || !strings.Contains(output, "testName=\"Dice microservice jaeger test\"") {
		t.Fatalf("expected output to contain passed message, got: %q", output)
	}
}

func TestRunTests_MultipleTests_DisplayTable(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-test-multiple")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	// Set up mock HTTP server
	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		// Verify query is selecting correct traceId
		if strings.Contains(sqlQuery, "trace_id = '1234567890abcdef1234567890abcdef'") {
			hits := []map[string]any{
				{
					"span_id":                  "span1",
					"reference_parent_span_id": "",
					"service_name":             "test-service",
					"operation_name":           "test-op",
					"span_kind":                "server",
					"span_status":              "ok",
					"start_time":               float64(1717495000000000000),
					"end_time":                 float64(1717495005000000000),
					"duration":                 float64(5000000000),
				},
			}
			return hits, http.StatusOK
		}

		return nil, http.StatusNotFound
	})

	// Setup YAML test definitions
	test1Content := `
name: "Dice microservice test 1"
description: "A quick test 1"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
waitBeforeFetch: 1ms
expectedTraces:
  - spans:
      - serviceName: "test-service"
        operationName: "test-op"
        spanKind: "server"
        spanStatus: "ok"
lastSpan:
  serviceName: "test-service"
  operationName: "test-op"
  spanKind: "server"
  spanStatus: "ok"
timeout: 100ms
retryDelay: 10ms
`

	test2Content := `
name: "Dice microservice test 2"
description: "A quick test 2"
trigger:
  type: "traceid"
  args:
    traceId: "abcdef1234567890abcdef1234567890"
waitBeforeFetch: 1ms
timeout: 50ms
retryDelay: 10ms
`

	testutils.CreateTempYAMLFile(t, tempDir, "test1.mt.yaml", test1Content)
	testutils.CreateTempYAMLFile(t, tempDir, "test2.mt.yaml", test2Content)

	// Save/Restore globals
	origConfig := cmd.Config

	cmd.Config = configuration.AppConfig{
		BackendType: "openobserve",
		Directory:   tempDir,
		Verbose:     false,
		OpenObserveConfig: &openobserve.OpenObserveConfig{
			BaseURL:    mockServer.URL,
			OrgName:    "default",
			StreamName: "default",
			Username:   "admin@example.com",
			Password:   "admin",
		},
	}

	defer func() {
		cmd.Config = origConfig
	}()

	var runErr error
	output := testutils.CaptureStdout(t, func() {
		runErr = cmd.RunTests(nil, nil)
	})

	if runErr != nil {
		t.Fatalf("expected run to succeed, got error: %v", runErr)
	}

	// Verify that output contains the table structure and content
	expectedHeaders := []string{"TEST NAME", "STATUS", "DETAILS"}
	for _, h := range expectedHeaders {
		if !strings.Contains(output, h) {
			t.Errorf("expected output to contain table header %q, got: %q", h, output)
		}
	}

	// Verify test 1 is in the table as PASSED
	if !strings.Contains(output, "Dice microservice test 1") {
		t.Errorf("expected output to contain test 1 name, got: %q", output)
	}
	if !strings.Contains(output, "PASSED") {
		t.Errorf("expected output to contain 'PASSED' status, got: %q", output)
	}

	// Verify test 2 is in the table as FAILED
	if !strings.Contains(output, "Dice microservice test 2") {
		t.Errorf("expected output to contain test 2 name, got: %q", output)
	}
	if !strings.Contains(output, "FAILED") {
		t.Errorf("expected output to contain 'FAILED' status, got: %q", output)
	}
}

func TestRunTests_Quiet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtrace-run-quiet")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	test1Content := `
name: "Dice microservice test 1"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
waitBeforeFetch: 1ms
timeout: 100ms
retryDelay: 10ms
`
	testutils.CreateTempYAMLFile(t, tempDir, "test1.mt.yaml", test1Content)

	// Mock server that returns valid Jaeger/OpenObserve trace response
	mockServer := testutils.StartMockOpenObserveServer(t, func(sqlQuery string) ([]map[string]any, int) {
		hits := []map[string]any{
			{
				"span_id":                  "s1",
				"reference_parent_span_id": "",
				"service_name":             "service-1",
				"operation_name":           "op-1",
				"span_kind":                "server",
				"span_status":              "ok",
				"start_time":               float64(1717590000000000000),
				"end_time":                 float64(1717590000100000000),
				"duration":                 float64(100000000),
			},
		}
		return hits, http.StatusOK
	})
	defer mockServer.Close()

	origConfig := cmd.Config
	cmd.Config = configuration.AppConfig{
		BackendType: "openobserve",
		Directory:   tempDir,
		Quiet:       true, // Enable quiet mode
		OpenObserveConfig: &openobserve.OpenObserveConfig{
			BaseURL:    mockServer.URL,
			OrgName:    "default",
			StreamName: "default",
			Username:   "admin@example.com",
			Password:   "admin",
		},
	}
	defer func() {
		cmd.Config = origConfig
	}()

	var runErr error
	output := testutils.CaptureStdoutWithLevel(t, slog.LevelWarn, func() {
		runErr = cmd.RunTests(nil, nil)
	})

	if runErr != nil {
		t.Fatalf("expected run to succeed, got error: %v", runErr)
	}

	// Verify that output does NOT contain the table headers and summary content
	if strings.Contains(output, "TEST NAME") || strings.Contains(output, "STATUS") || strings.Contains(output, "DETAILS") {
		t.Errorf("expected summary table to be suppressed, but got: %q", output)
	}
}
