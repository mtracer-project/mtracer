/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/export"
	"github.com/mtracer-project/mtracer/trigger"

	"github.com/spf13/cobra"
)

const DEFAULT_TRIGGER_EXAMPLE = `trigger:
  type: "http" # http | traceId | nats | jetstream | gRPC
  args:
    url: "http://example.com/api/endpoint"
    method: "POST"
    headers:
      - Content-Type: "application/json"
      - Authorization: "Bearer <token>"
    body: '{"key": "value"}'`

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new test case",
	Long:  `Create a new test case with the specified name. This command will generate a new test case starting from a template file.`,
	Run:   CreateTestCase,
}

var TriggerType string

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&TriggerType, "trigger-type", "t", "http", "Type of trigger for the test case. Supported types: http, traceId, nats, jetstream, playwright, gRPC")
}

func CreateTestCase(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		slog.Error(fmt.Sprintf("Please provide a name for the test case. Usage: %s create <test-name>", domain.CLI_NAME))
		return
	}

	baseDir := Config.Directory
	if baseDir == "" {
		baseDir = "." // current directory as default
	}

	testName := args[0]
	testFileName := fmt.Sprintf("%s.mt.yaml", testName)
	testFilePath := fmt.Sprintf("%s/%s", baseDir, testFileName)

	if _, err := os.Stat(testFilePath); err == nil {
		slog.Error("A test case with that name already exists. Please choose a different name.", "testName", testName, "testFilePath", testFilePath)
		return
	}

	triggerExample := DEFAULT_TRIGGER_EXAMPLE
	trigger, err := trigger.NewTriggerFromType(TriggerType)
	if err == nil {
		triggerExample = trigger.Example()
	}

	templateContent := `
name: "%s"
description: "Description of the test case"
setupCommands:
  - type: "shell"
    cmd: "echo 'Setting up test environment'"
    cleanupCmd:
      type: "shell"
      cmd: "echo 'Cleaning up test environment'"
%s
waitBeforeFetch: 5s
timeout: 60s
retryDelay: 1s
expectedProperties:
  maxDuration: 30s
  minDuration: 1s
  spanCount: 5
  errorCount: 0
expectedTraces:
  - ordered: yes # yes | no
    checker: contains # contains | strict | startsWith | endsWith
    spans:
      - serviceName: "example-service"
      - serviceName: "example2-service"
        operationName: "destroyed-everything"
        spanKind: "internal"
        spanStatus: "unset"
assertions:
  - name: "Check error count and span status"
    type: "cel"
    queries:
      errorCheck: "trace.errorCount == 0"
      spanStatusCheck: "trace.spans.all(s, s.spanStatus == 'UNSET')"
lastSpan:
  serviceName: "example2-service"
  operationName: "destroyed-everything"
  spanKind: "internal"
  spanStatus: "unset"
`

	formattedContent := fmt.Sprintf(templateContent, testName, triggerExample)

	err = os.WriteFile(testFilePath, []byte(formattedContent), 0o600)
	if err != nil {
		slog.Error("Error creating test case file", "error", err)
		return
	}

	slog.Info("Test case %screated%s successfully", "testName", testName, "testFilePath", testFilePath)
	if !Config.Quiet {
		fmt.Printf("Test %s case %screated%s successfully at %s", testName, export.COLOR_GREEN, export.COLOR_RESET, testFilePath)
	}
}
