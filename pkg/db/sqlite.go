package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"

	"github.com/medyagh/gopogh/pkg/models"
	_ "modernc.org/sqlite" // Blank import used for registering SQLite driver as a database driver
)

var createEnvironmentTestsTableSQL = `
	CREATE TABLE IF NOT EXISTS db_environment_tests (
		CommitID TEXT,
		EnvName TEXT,
		GopoghTime TEXT,
		TestTime TEXT,
		NumberOfFail INTEGER,
		NumberOfPass INTEGER,
		NumberOfSkip INTEGER,
		TotalDuration REAL,
		GopoghVersion TEXT,
		PRIMARY KEY (CommitID, EnvName)
	);
`
var createTestCasesTableSQL = `
	CREATE TABLE IF NOT EXISTS db_test_cases (
		PR TEXT,
		CommitId TEXT,
		TestName TEXT,
		Result TEXT,
		Duration REAL,
		EnvName TEXT,
		TestOrder INTEGER,
		PRIMARY KEY (CommitId, EnvName, TestName)
	);
`

type sqlite struct {
	db   *sqlx.DB
	path string
}

// Set adds/updates rows to the database
func (m *sqlite) Set(commitRow models.DBEnvironmentTest, dbRows []models.DBTestCase) error {
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

	sqlInsert := `INSERT OR REPLACE INTO db_test_cases (PR, CommitId, TestName, Result, Duration, EnvName, TestOrder) VALUES (?, ?, ?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		return fmt.Errorf("failed to prepare SQL insert statement: %v", err)
	}
	defer stmt.Close()

	for _, r := range dbRows {
		_, err := stmt.Exec(r.PR, r.CommitID, r.TestName, r.Result, r.Duration, r.EnvName, r.TestOrder)
		if err != nil {
			return fmt.Errorf("failed to execute SQL insert: %v", err)
		}
	}

	sqlInsert = `INSERT OR REPLACE INTO db_environment_tests (CommitID, EnvName, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration, GopoghVersion) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.Exec(sqlInsert, commitRow.CommitID, commitRow.EnvName, commitRow.GopoghTime, commitRow.TestTime, commitRow.NumberOfFail, commitRow.NumberOfPass, commitRow.NumberOfSkip, commitRow.TotalDuration, commitRow.GopoghVersion)
	if err != nil {
		return fmt.Errorf("failed to execute SQL insert: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit SQL insert transaction: %v", err)
	}
	return rollbackError
}

// newSQLite opens the database returning an SQLite database struct instance
func newSQLite(cfg config) (*sqlite, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	database, err := sqlx.Connect("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}
	m := &sqlite{
		db:   database,
		path: cfg.Path,
	}
	return m, nil
}

// Initialize creates the tables within the SQLite database
func (m *sqlite) Initialize() error {

	if _, err := m.db.Exec(createEnvironmentTestsTableSQL); err != nil {
		return fmt.Errorf("failed to initialize environment tests table: %v", err)
	}
	if _, err := m.db.Exec(createTestCasesTableSQL); err != nil {
		return fmt.Errorf("failed to initialize test cases table: %v", err)
	}
	return nil
}
