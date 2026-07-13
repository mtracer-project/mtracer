/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/moby/moby/client"
	"github.com/mtracer-project/mtracer/export"
	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"
	"github.com/mtracer-project/mtracer/test"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check the syntax of a test case file",
	Long:  `Check the syntax of a test case file by validating its structure and content.`,
	RunE:  CheckTestCase,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func CheckTestCase(cmd *cobra.Command, args []string) error {
	testPaths, err := getTestPaths(args)
	if err != nil {
		return err
	}

	dtos, err := parser.ParseTests(testPaths)
	if err != nil {
		return err
	}

	var ctx context.Context
	if cmd != nil {
		ctx = cmd.Context()
	} else {
		ctx = context.Background()
	}

	traceAdapter, err := Config.NewTraceAdapterFromConfig(ctx)
	if err != nil {
		return fmt.Errorf("error creating trace adapter: %w", err)
	}

	idGenerator := &idgenerator.IdGeneratorV1{}

	cli, err := client.New(
		client.FromEnv,
	)
	if err != nil {
		return fmt.Errorf("error creating Docker client: %w", err)
	}

	var suitesList []*test.TestSuite

	for _, dto := range dtos {
		dtoPath := filepath.Dir(dto.FilePath)
		_, err := test.NewTraceTest(dto, idGenerator, cli, traceAdapter, test.TraceTestOptions{}, dtoPath, ctx)
		var res *test.TestResult
		if err != nil {
			slog.Info("Test case file is invalid", "testName", dto.Name, "error", err)

			res = &test.TestResult{
				Passed: false,
				Args:   []any{"message", "Test case file is invalid", "error", err.Error()},
			}
		} else {
			slog.Info("Test case file is valid", "testName", dto.Name)
			res = &test.TestResult{
				Passed: true,
			}
		}
		suitesList = append(suitesList, test.NewTestSuite(
			dto.Name,
			[]*test.TestResult{res},
		))
	}

	if !Config.Quiet {
		err = export.DisplayTestsSummary(suitesList, "VALID", "INVALID")
		if err != nil {
			slog.Error("error displaying the tests summary", "error", err)
		}
	}

	return nil
}
