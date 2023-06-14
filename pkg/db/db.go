package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // Blank import used for registering SQLite driver as a database driver
	"github.com/medyagh/gopogh/pkg/models"
)

// PopulateDatabase adds/updates rows to the database
func PopulateDatabase(database *sql.DB, dbRows []models.DatabaseTestRow, commitRow models.DatabaseCommitRow) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("failed to create SQL transaction: %v", err)
	}

	var rollbackError error
	defer func() {
		if rErr := tx.Rollback(); rErr != nil {
			rollbackError = fmt.Errorf("error occurred during rollback: %v", rErr)
		}
	}()

	sqlInsert := `INSERT OR REPLACE INTO tests (PR, CommitId, TestName, Result) VALUES (?, ?, ?, ?)`
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

	sqlInsert = `INSERT OR REPLACE INTO commits (CommitID, EnvName, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip) VALUES (?, ?, ?, ?, ?, ?, ?)`
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

// CreateDatabase opens the database and creates the tables if not present
func CreateDatabase(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS tests (
		PR TEXT,
		CommitId TEXT,
		TestName TEXT,
		Result TEXT,
		PRIMARY KEY (CommitId, TestName)
	);
`

	_, err = database.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create tests table: %v", err)
	}

	createTableSQL = `
	CREATE TABLE IF NOT EXISTS commits (
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
	_, err = database.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create commits table: %v", err)
	}

	return database, nil
}
