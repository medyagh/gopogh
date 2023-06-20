package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3" // Blank import used for registering SQLite driver as a database driver
	"github.com/medyagh/gopogh/pkg/models"
)

var createEnviornmentTestsTableSQL = `
	CREATE TABLE IF NOT EXISTS db_enviornment_tests (
		CommitID TEXT,
    	EnvName TEXT,
    	GopoghTime TEXT,
    	TestTime TEXT,
    	NumberOfFail INTEGER,
    	NumberOfPass INTEGER,
    	NumberOfSkip INTEGER,
		PRIMARY KEY (CommitID)
	);
`
var createTestCasesTableSQL = `
	CREATE TABLE IF NOT EXISTS db_test_cases (
		PR TEXT,
		CommitId TEXT,
		TestName TEXT,
		Result TEXT,
		PRIMARY KEY (CommitId, TestName)
	);
`

type SQLite struct {
	db   *sqlx.DB
	path string
}

// Set adds/updates rows to the database
func (m *SQLite) Set(commitRow models.DbEnvironmentTest, dbRows []models.DbTestCase) error {
	tx, err := m.db.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to create SQL transaction: %v", err)
	}

	var rollbackError error
	defer func() {
		if rErr := tx.Rollback(); rErr != nil {
			rollbackError = fmt.Errorf("error occurred during rollback: %v", rErr)
		}
	}()

	sqlInsert := `INSERT OR REPLACE INTO db_test_cases (PR, CommitId, TestName, Result) VALUES (?, ?, ?, ?)`
	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		return fmt.Errorf("failed to prepare SQL insert statement: %v", err)
	}
	defer stmt.Close()

	for _, r := range dbRows {
		_, err := stmt.Exec(r.PR, r.CommitID, r.TestName, r.Result)
		if err != nil {
			return fmt.Errorf("failed to execute SQL insert: %v", err)
		}
	}

	sqlInsert = `INSERT OR REPLACE INTO db_enviornment_tests (CommitID, EnvName, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.Exec(sqlInsert, commitRow.CommitID, commitRow.EnvName, commitRow.GopoghTime, commitRow.TestTime, commitRow.NumberOfFail, commitRow.NumberOfPass, commitRow.NumberOfSkip)
	if err != nil {
		return fmt.Errorf("failed to execute SQL insert: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit SQL insert transaction: %v", err)
	}
	return rollbackError
}

// NewSQLite opens the database returning an SQLite database struct instance
func NewSQLite(cfg Config) (*SQLite, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	database, err := sqlx.Connect("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}
	m := &SQLite{
		db:   database,
		path: cfg.Path,
	}
	return m, nil
}

// Initialize creates the tables within the SQLite database
func (m *SQLite) Initialize() error {

	if _, err := m.db.Exec(createEnviornmentTestsTableSQL); err != nil {
		return fmt.Errorf("failed to initialize enviornment tests table: %w", err)
	}
	if _, err := m.db.Exec(createTestCasesTableSQL); err != nil {
		return fmt.Errorf("failed to initialize test cases table: %w", err)
	}
	return nil
}
