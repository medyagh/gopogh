package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Blank import used for registering postgres driver as a database driver
	"github.com/medyagh/gopogh/pkg/models"
)

var pgEnvTableSchema = `
	CREATE TABLE IF NOT EXISTS db_environment_tests (
		CommitID TEXT,
    	EnvName TEXT,
    	GopoghTime TEXT,
    	TestTime TEXT,
    	NumberOfFail INTEGER,
    	NumberOfPass INTEGER,
    	NumberOfSkip INTEGER,
		PRIMARY KEY (CommitID, EnvName)
	);
`
var pgTestCasesTableSchema = `
	CREATE TABLE IF NOT EXISTS db_test_cases (
		PR TEXT,
		CommitId TEXT,
		TestName TEXT,
		Result TEXT,
		PRIMARY KEY (CommitId, EnvName, TestName)
	);
`

type Postgres struct {
	db   *sqlx.DB
	path string
}

// Set adds/updates rows to the database
func (m *Postgres) Set(commitRow models.DBEnvironmentTest, dbRows []models.DBTestCase) error {
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

	sqlInsert := `
		INSERT INTO db_test_cases (PR, CommitId, TestName, Result)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (CommitId, TestName)
		DO UPDATE SET Result = excluded.Result 
	`
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

	sqlInsert = `
		INSERT INTO db_environment_tests (CommitID, EnvName, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (CommitId, EnvName)
		DO UPDATE SET (GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip) = (EXCLUDED.GopoghTime, EXCLUDED.TestTime, EXCLUDED.NumberOfFail, EXCLUDED.NumberOfPass, EXCLUDED.NumberOfSkip)
		`
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

// newPostgres opens the database returning a Postgres database struct instance
func newPostgres(cfg config) (*Postgres, error) {
	database, err := sqlx.Connect("postgres", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}
	m := &Postgres{
		db:   database,
		path: cfg.Path,
	}
	return m, nil
}

// Initialize creates the tables within the Postgres database
func (m *Postgres) Initialize() error {
	if _, err := m.db.Exec(pgEnvTableSchema); err != nil {
		return fmt.Errorf("failed to initialize environment tests table: %v", err)
	}
	if _, err := m.db.Exec(pgTestCasesTableSchema); err != nil {
		return fmt.Errorf("failed to initialize test cases table: %v", err)
	}
	return nil
}
