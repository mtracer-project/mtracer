package postexecchecks

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mtrace-project/mtrace/parser"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
	// Update the switch case in NewSQLPostExecCheck if you add a new driver
)

const (
	DEFAULT_SQL_DRIVER_NAME = "pgx"
)

type SQLPostExecCheck struct {
	query      string
	driverName string
	dsn        string

	ctx context.Context
}

func (s *SQLPostExecCheck) Check() (bool, error) {
	db, err := sql.Open(s.driverName, s.dsn)
	if err != nil {
		return false, fmt.Errorf("error opening db: %w", err)
	}
	defer db.Close() // nolint:errcheck

	if err := db.PingContext(s.ctx); err != nil {
		return false, fmt.Errorf("error reaching db: %w", err)
	}

	wrapperQuery := fmt.Sprintf("SELECT CASE WHEN (%s) THEN 1 ELSE 0 END", s.query)

	// Special handling for Oracle: if the query does not have a FROM clause, we need to add "FROM dual"
	if s.driverName == "oracle" {
		wrapperQuery += " FROM dual"
	}

	var result int
	// Execute the query passing the context to handle potential timeouts
	err = db.QueryRowContext(s.ctx, wrapperQuery).Scan(&result)
	if err != nil {
		return false, fmt.Errorf("error during SQL check execution: %w", err)
	}

	// Return true if the result is 1, otherwise false
	return result == 1, nil
}

func NewSQLPostExecCheck(dto *parser.PostExecCheckDTO, ctx context.Context) (*SQLPostExecCheck, error) {
	if dto.Args == nil {
		return nil, fmt.Errorf("args are required for sql post exec check")
	}

	query, ok := dto.Args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required for sql post exec check")
	}

	driverName, ok := dto.Args["driverName"].(string)
	if !ok {
		driverName = DEFAULT_SQL_DRIVER_NAME
	}
	switch driverName {
	case "mysql", "pgx", "sqlite":
		// Supported drivers
	default:
		return nil, fmt.Errorf("unsupported driver name: %s", driverName)
	}

	dsn, ok := dto.Args["dsn"].(string)
	if !ok {
		return nil, fmt.Errorf("dsn is required for sql post exec check")
	}

	return &SQLPostExecCheck{
		query:      query,
		driverName: driverName,
		dsn:        dsn,
		ctx:        ctx,
	}, nil
}
