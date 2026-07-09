package postexecchecks_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	postexecchecks "github.com/mtrace-project/mtrace/postExecChecks"

	_ "modernc.org/sqlite"
)

func TestNewSQLPostExecCheck(t *testing.T) {
	t.Run("nil args returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check",
			Type: "sql",
			Args: nil,
		}
		_, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err == nil || !strings.Contains(err.Error(), "args are required") {
			t.Errorf("expected 'args are required' error, got: %v", err)
		}
	})

	t.Run("missing query returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check",
			Type: "sql",
			Args: map[string]any{
				"dsn": "postgres://localhost:5432/db",
			},
		}
		_, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err == nil || !strings.Contains(err.Error(), "query is required") {
			t.Errorf("expected 'query is required' error, got: %v", err)
		}
	})

	t.Run("missing dsn returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check",
			Type: "sql",
			Args: map[string]any{
				"query": "SELECT 1 = 1",
			},
		}
		_, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err == nil || !strings.Contains(err.Error(), "dsn is required") {
			t.Errorf("expected 'dsn is required' error, got: %v", err)
		}
	})

	t.Run("unsupported driver returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check",
			Type: "sql",
			Args: map[string]any{
				"query":      "SELECT 1 = 1",
				"dsn":        "postgres://localhost:5432/db",
				"driverName": "oracle", // Unsupported driver
			},
		}
		_, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err == nil || !strings.Contains(err.Error(), "unsupported driver name") {
			t.Errorf("expected 'unsupported driver name' error, got: %v", err)
		}
	})

	t.Run("valid construction with explicit driver", func(t *testing.T) {
		for _, driver := range []string{"mysql", "pgx", "sqlite"} {
			t.Run(driver, func(t *testing.T) {
				dto := &parser.PostExecCheckDTO{
					Name: "sql-check-" + driver,
					Type: "sql",
					Args: map[string]any{
						"query":      "SELECT 1 = 1",
						"dsn":        "test-dsn",
						"driverName": driver,
					},
				}
				check, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
				if err != nil {
					t.Fatalf("unexpected error for driver %s: %v", driver, err)
				}
				if check == nil {
					t.Fatalf("expected non-nil check for driver %s", driver)
				}
			})
		}
	})

	t.Run("default driver is pgx", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check-default",
			Type: "sql",
			Args: map[string]any{
				"query": "SELECT 1 = 1",
				"dsn":   "test-dsn",
			},
		}
		check, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if check == nil {
			t.Fatal("expected non-nil check")
		}
	})

	t.Run("query with wrong type returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check",
			Type: "sql",
			Args: map[string]any{
				"query": 123,
				"dsn":   "test-dsn",
			},
		}
		_, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err == nil || !strings.Contains(err.Error(), "query is required") {
			t.Errorf("expected 'query is required' error, got: %v", err)
		}
	})
}

func TestSQLPostExecCheck_Check(t *testing.T) {
	t.Run("query evaluates to true", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check-true",
			Type: "sql",
			Args: map[string]any{
				"query":      "1 = 1",
				"dsn":        ":memory:",
				"driverName": "sqlite",
			},
		}
		check, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		passed, err := check.Check()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !passed {
			t.Errorf("expected test to pass")
		}
	})

	t.Run("query evaluates to false", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check-false",
			Type: "sql",
			Args: map[string]any{
				"query":      "1 = 0",
				"dsn":        ":memory:",
				"driverName": "sqlite",
			},
		}
		check, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		passed, err := check.Check()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if passed {
			t.Errorf("expected test to fail")
		}
	})

	t.Run("invalid driver or dsn returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check-invalid-driver",
			Type: "sql",
			Args: map[string]any{
				"query":      "1 = 1",
				"dsn":        "invalid-connection-string",
				"driverName": "mysql", // mysql driver with invalid dsn
			},
		}
		check, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = check.Check()
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("invalid query returns error", func(t *testing.T) {
		dto := &parser.PostExecCheckDTO{
			Name: "sql-check-invalid-query",
			Type: "sql",
			Args: map[string]any{
				"query":      "SELECT * FROM nonexistent_table",
				"dsn":        ":memory:",
				"driverName": "sqlite",
			},
		}
		check, err := postexecchecks.NewSQLPostExecCheck(dto, context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = check.Check()
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}
