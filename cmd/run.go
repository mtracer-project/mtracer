/*
Copyright © 2026 NAME HERE alessandro.dinato@gmail.com
*/
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/mtracer-project/mtracer/analytics"
	"github.com/mtracer-project/mtracer/export"
	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"
	"github.com/mtracer-project/mtracer/test"

	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

var (
	ExportFormats    []string
	AnalyticsFormats []string
	RunCount         int
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run every *.mt.yaml file or a specific test in the specified directory",
	Long: `Run every *.mt.yaml test in the specified directory.
	It is also possible to run specific tests by providing their file names as arguments.
	The directory that you run the command in is the default one.
	This command will look for all files with the .mt.yaml extension in the specified directory and execute them as tests.
	`,
	RunE: RunTests,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringSliceVarP(&ExportFormats, "export-to", "e", []string{}, "Export formats for the test results. If not specified, no export will be performed. Supported formats: json, junit, markdown/md")
	runCmd.Flags().StringSliceVarP(&AnalyticsFormats, "analytics", "a", []string{}, "Get analytics on the generated traces and export them to the specified formats. If not specified, no analytics will be performed. Supported formats: html, json")
	runCmd.Flags().IntVar(&RunCount, "count", 1, "Number of times to execute each test")
}

func RunTests(cmd *cobra.Command, args []string) error {
	if RunCount < 1 {
		return fmt.Errorf("--count flag has to be greater or equal to 1, got: %d", RunCount)
	}

	analyticsEnabled := len(AnalyticsFormats) > 0

	var ctx context.Context
	if cmd != nil {
		ctx = cmd.Context()
	} else {
		ctx = context.Background()
	}

	// Get the paths of the test files to run
	testPaths, err := getTestPaths(args)
	if err != nil {
		return fmt.Errorf("error getting test paths: %w", err)
	}

	// Parse the test files into DTOs
	dtos, err := parser.ParseTests(testPaths)
	if err != nil {
		return err
	}

	// Create the adapter for the observability backend based on the configuration
	traceAdapter, err := Config.NewTraceAdapterFromConfig(ctx)
	if err != nil {
		return fmt.Errorf("error creating trace adapter: %w", err)
	}

	// Set up exporters
	var formats []string
	formats = append(formats, ExportFormats...)
	if !Config.Quiet {
		formats = append(formats, export.STDOUT_FORMAT)
	}
	exporters, err := export.NewExporters(formats, RunCount, Config.Directory)
	if err != nil {
		return fmt.Errorf("error creating exporters: %w", err)
	}

	// Set up analytics exporters
	analyticsExporters, err := analytics.NewAnalyticsExporters(AnalyticsFormats, Config.Directory)
	if err != nil {
		return fmt.Errorf("error creating analytics exporters: %w", err)
	}

	// Create an ID generator for generating IDs for span and trace
	idGenerator := &idgenerator.IdGeneratorV1{}

	// Create a Docker client for interacting with Docker containers
	cli, err := client.New(
		client.FromEnv,
	)
	if err != nil {
		return fmt.Errorf("error creating Docker client: %w", err)
	}

	var suites []*test.TestSuite

	traceTestOpts := test.TraceTestOptions{
		CollectTrace: analyticsEnabled,
	}
	// Parse and run each test, collecting results
	for _, dto := range dtos {
		dtoPath := filepath.Dir(dto.FilePath)
		var t test.Test
		t, err := test.NewTraceTest(dto, idGenerator, cli, traceAdapter, traceTestOpts, dtoPath, ctx)
		if err != nil {
			suites = append(suites, test.NewTestSuite(dto.Name, []*test.TestResult{{
				Passed:   false,
				Duration: 0,
				Args:     []any{"message", "Error creating the test", "error", err.Error()},
			}}))
			slog.Warn("error creating the test, skipping it", "testName", dto.Name, "error", err.Error())
			continue
		}

		var testResults []*test.TestResult
		// Run the test the specified number of times
		for i := 0; i < RunCount; i++ {
			result := t.Run()
			testResults = append(testResults, result)
			resultArgs := []any{"testName", dto.Name}

			if result.Passed {
				slog.Info("PASSED", resultArgs...)
			} else {
				slog.Info("FAILED", resultArgs...)
			}
		}
		suites = append(suites, test.NewTestSuite(
			dto.Name,
			testResults,
		))
	}

	// Export results to the specified formats
	for _, exporter := range exporters {
		err = exporter.Export(suites)
		if err != nil {
			slog.Error("error exporting test results", "format", exporter.Format(), "error", err)
		}
	}

	if analyticsEnabled {
		slog.Info("Analytics enabled, calculating metrics...")
		testAnalytics := analytics.Build(suites)

		// Export analytics results to the specified formats
		for _, analyticsExporter := range analyticsExporters {
			err = analyticsExporter.Export(testAnalytics)
			if err != nil {
				slog.Error("error exporting analytics results", "format", analyticsExporter.Format(), "error", err)
			}
		}
	}

	return nil
}
