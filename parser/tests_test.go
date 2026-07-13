package parser_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/parser"
	testutils "github.com/mtracer-project/mtracer/testUtils"
)

func TestParseTests_EmptyFilePaths(t *testing.T) {
	_, err := parser.ParseTests(nil)
	if err == nil || !strings.Contains(err.Error(), "no .mt.yaml files found to execute") {
		t.Errorf("expected empty filePaths error, got: %v", err)
	}

	_, err = parser.ParseTests([]string{})
	if err == nil || !strings.Contains(err.Error(), "no .mt.yaml files found to execute") {
		t.Errorf("expected empty filePaths error, got: %v", err)
	}
}

func TestParseTests_NonExistentFile(t *testing.T) {
	_, err := parser.ParseTests([]string{"/nonexistent/file.mt.yaml"})
	if err == nil || !strings.Contains(err.Error(), "error opening file") {
		t.Errorf("expected open error, got: %v", err)
	}
}

func TestParseTests_MalformedYAML(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-parser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	filePath := testutils.CreateTempYAMLFile(t, tempDir, "bad.mt.yaml", `
name: "Malformed yaml
trigger:
`)

	_, err = parser.ParseTests([]string{filePath})
	if err == nil || !strings.Contains(err.Error(), "error parsing YAML file") {
		t.Errorf("expected yaml parsing error, got: %v", err)
	}
}

func TestParseTests_EmptyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-parser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	filePath := testutils.CreateTempYAMLFile(t, tempDir, "empty.mt.yaml", "")

	_, err = parser.ParseTests([]string{filePath})
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestParseTests_CommentsOnlyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-parser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	filePath := testutils.CreateTempYAMLFile(t, tempDir, "comments.mt.yaml", "# only comment")

	_, err = parser.ParseTests([]string{filePath})
	if err == nil {
		t.Fatal("expected error for comments-only file, got nil")
	}
}

func TestParseTests_ValidationError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-parser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	// Missing trigger which is required
	filePath := testutils.CreateTempYAMLFile(t, tempDir, "invalid.mt.yaml", `
name: "Invalid Test"
`)

	_, err = parser.ParseTests([]string{filePath})
	if err == nil || !strings.Contains(err.Error(), "validation error in file") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestParseTests_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtracer-parser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // nolint:errcheck

	yamlContent := `
name: "Successful Parse Test"
description: "A test to verify parser functionality"
trigger:
  type: "traceid"
  args:
    traceId: "1234567890abcdef1234567890abcdef"
expectedTraces:
  - spans:
      - serviceName: "service-1"
        operationName: "op-1"
        spanKind: "server"
        spanStatus: "ok"
lastSpan:
  serviceName: "service-1"
`

	filePath := testutils.CreateTempYAMLFile(t, tempDir, "success.mt.yaml", yamlContent)

	tests, err := parser.ParseTests([]string{filePath})
	if err != nil {
		t.Fatalf("unexpected error parsing valid file: %v", err)
	}

	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}

	got := tests[0]
	if got.Name != "Successful Parse Test" {
		t.Errorf("expected name 'Successful Parse Test', got %q", got.Name)
	}
	if got.Description != "A test to verify parser functionality" {
		t.Errorf("expected description 'A test to verify parser functionality', got %q", got.Description)
	}
	if got.Trigger == nil || got.Trigger.Type != "traceid" {
		t.Errorf("expected trigger type 'traceid', got %v", got.Trigger)
	}
	if got.Trigger.Args["traceId"] != "1234567890abcdef1234567890abcdef" {
		t.Errorf("expected traceId arg '1234567890abcdef1234567890abcdef', got %v", got.Trigger.Args["traceId"])
	}

	// Verify defaults set during validation
	if got.Timeout == nil || time.Duration(*got.Timeout) != 60*time.Second {
		t.Errorf("expected default timeout 60s, got %v", got.Timeout)
	}

	if len(got.ExpectedTraces) != 1 {
		t.Fatalf("expected 1 expected trace, got %d", len(got.ExpectedTraces))
	}

	expTrace := got.ExpectedTraces[0]
	if expTrace.Checker == nil || *expTrace.Checker != "contains" {
		t.Errorf("expected default checker 'contains', got %v", expTrace.Checker)
	}
	if len(expTrace.Spans) != 1 {
		t.Fatalf("expected 1 span in expected trace, got %d", len(expTrace.Spans))
	}

	span := expTrace.Spans[0]
	if span.ServiceName != "service-1" {
		t.Errorf("expected serviceName 'service-1', got %q", span.ServiceName)
	}
	if span.OperationName == nil || *span.OperationName != "op-1" {
		t.Errorf("expected operationName 'op-1', got %v", span.OperationName)
	}

	if got.LastSpan == nil || got.LastSpan.ServiceName != "service-1" {
		t.Errorf("expected lastSpan serviceName 'service-1', got %v", got.LastSpan)
	}
}
