package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/medyagh/gopogh/pkg/models"
	_ "modernc.org/sqlite" // Blank import used for registering SQLite driver as a database driver
)

var createEnvironmentTestsTableSQL = `
	CREATE TABLE IF NOT EXISTS db_environment_tests (
		CommitID TEXT,
		EnvName TEXT,
		EnvGroup TEXT NOT NULL DEFAULT 'Legacy',
		GopoghTime TEXT,
		TestTime TEXT,
		NumberOfFail INTEGER,
		NumberOfPass INTEGER,
		NumberOfSkip INTEGER,
		TotalDuration REAL,
		GopoghVersion TEXT,
		ArtifactPath TEXT NOT NULL DEFAULT '',
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
		TestTime TEXT,
		PRIMARY KEY (CommitId, EnvName, TestName)
	);
`

type sqlite struct {
	db   *sqlx.DB
	path string
}

// Set adds/updates rows to the database
func (m *sqlite) Set(commitRow models.DBEnvironmentTest, dbRows []models.DBTestCase) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to create SQL transaction: %v", err)
	}

	var rollbackError error
	defer func() {
		if rErr := tx.Rollback(); rErr != nil {
			rollbackError = fmt.Errorf("error occurred during rollback: %v", rErr)
		}
	}()

	sqlInsert := `INSERT OR REPLACE INTO db_test_cases (PR, CommitId, TestName, Result, Duration, EnvName, TestOrder, TestTime) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		return fmt.Errorf("failed to prepare SQL insert statement: %v", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, r := range dbRows {
		_, err := stmt.Exec(r.PR, r.CommitID, r.TestName, r.Result, r.Duration, r.EnvName, r.TestOrder, r.TestTime.String())
		if err != nil {
			return fmt.Errorf("failed to execute SQL insert: %v", err)
		}
	}

	envGroup := strings.TrimSpace(commitRow.EnvGroup)
	if envGroup == "" {
		envGroup = "Legacy"
	}
	sqlInsert = `INSERT OR REPLACE INTO db_environment_tests (CommitID, EnvName, EnvGroup, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration, GopoghVersion, ArtifactPath) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.Exec(sqlInsert, commitRow.CommitID, commitRow.EnvName, envGroup, commitRow.GopoghTime, commitRow.TestTime.String(), commitRow.NumberOfFail, commitRow.NumberOfPass, commitRow.NumberOfSkip, commitRow.TotalDuration, commitRow.GopoghVersion, commitRow.ArtifactPath)
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
	if err := os.MkdirAll(filepath.Dir(cfg.path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	database, err := sqlx.Connect("sqlite", cfg.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}
	m := &sqlite{
		db:   database,
		path: cfg.path,
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
	if err := ensureSQLiteColumn(m.db, "db_environment_tests", "EnvGroup", "TEXT NOT NULL DEFAULT 'Legacy'"); err != nil {
		return fmt.Errorf("failed to ensure EnvGroup on environment tests table: %v", err)
	}
	if err := ensureSQLiteColumn(m.db, "db_environment_tests", "ArtifactPath", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return fmt.Errorf("failed to ensure ArtifactPath on environment tests table: %v", err)
	}
	if _, err := m.db.Exec(`UPDATE db_environment_tests SET ArtifactPath = '' WHERE ArtifactPath IS NULL;`); err != nil {
		return fmt.Errorf("failed to backfill ArtifactPath on environment tests table: %v", err)
	}
	if _, err := m.db.Exec(`UPDATE db_environment_tests SET EnvGroup = 'Legacy' WHERE EnvGroup IS NULL OR EnvGroup = '';`); err != nil {
		return fmt.Errorf("failed to backfill EnvGroup on environment tests table: %v", err)
	}
	return nil
}

func ensureSQLiteColumn(db *sqlx.DB, table, column, columnType string) error {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, column, columnType)
	_, err := db.Exec(query)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "duplicate column name") {
		return nil
	}
	return err
}

// GetEnvironmentTestsAndTestCases writes the database tables to a map with the keys environmentTests and testCases
// This is not yet supported for sqlite
func (m *sqlite) GetEnvironmentTestsAndTestCases() (map[string]interface{}, error) {
	return nil, nil
}

// GetEnvCharts writes the overall environment charts to a map with the keys recentFlakePercentTable, flakeRateByWeek, flakeRateByDay, and countsAndDurations
// This is not yet supported for sqlite
func (m *sqlite) GetEnvCharts(_ string, _ int) (map[string]interface{}, error) {
	return nil, nil
}

// GetTestCharts writes the individual test chart data to a map with the keys flakeByDay and flakeByWeek
// This is not yet supported for sqlite
func (m *sqlite) GetTestCharts(_ string, _ string) (map[string]interface{}, error) {
	return nil, nil
}

// GetOverview writes the overview charts to a map with the keys summaryAvgFail and summaryTable
// This is not yet supported for sqlite
func (m *sqlite) GetOverview(int) (map[string]interface{}, error) {
	return nil, nil
}
