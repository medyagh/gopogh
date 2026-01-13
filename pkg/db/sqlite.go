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
		PRIMARY KEY (CommitID, EnvName, EnvGroup)
	);
`
var createTestCasesTableSQL = `
	CREATE TABLE IF NOT EXISTS db_test_cases (
		PR TEXT,
		CommitId TEXT,
		EnvGroup TEXT NOT NULL DEFAULT 'Legacy',
		TestName TEXT,
		Result TEXT,
		Duration REAL,
		EnvName TEXT,
		TestOrder INTEGER,
		TestTime TEXT,
		PRIMARY KEY (CommitId, EnvName, EnvGroup, TestName)
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

	envGroup := strings.TrimSpace(commitRow.EnvGroup)
	if envGroup == "" {
		for _, row := range dbRows {
			rowEnvGroup := strings.TrimSpace(row.EnvGroup)
			if rowEnvGroup != "" {
				envGroup = rowEnvGroup
				break
			}
		}
	}
	if envGroup == "" {
		envGroup = "Legacy"
	}

	var rollbackError error
	defer func() {
		if rErr := tx.Rollback(); rErr != nil {
			rollbackError = fmt.Errorf("error occurred during rollback: %v", rErr)
		}
	}()

	sqlInsert := `INSERT OR REPLACE INTO db_test_cases (PR, CommitId, EnvGroup, TestName, Result, Duration, EnvName, TestOrder, TestTime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		return fmt.Errorf("failed to prepare SQL insert statement: %v", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, r := range dbRows {
		rowEnvGroup := strings.TrimSpace(r.EnvGroup)
		if rowEnvGroup == "" {
			rowEnvGroup = envGroup
		}
		_, err := stmt.Exec(r.PR, r.CommitID, rowEnvGroup, r.TestName, r.Result, r.Duration, r.EnvName, r.TestOrder, r.TestTime.String())
		if err != nil {
			return fmt.Errorf("failed to execute SQL insert: %v", err)
		}
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
	if err := ensureSQLiteColumn(m.db, "db_test_cases", "EnvGroup", "TEXT NOT NULL DEFAULT 'Legacy'"); err != nil {
		return fmt.Errorf("failed to ensure EnvGroup on test cases table: %v", err)
	}
	if _, err := m.db.Exec(`UPDATE db_environment_tests SET ArtifactPath = '' WHERE ArtifactPath IS NULL;`); err != nil {
		return fmt.Errorf("failed to backfill ArtifactPath on environment tests table: %v", err)
	}
	if _, err := m.db.Exec(`UPDATE db_environment_tests SET EnvGroup = 'Legacy' WHERE EnvGroup IS NULL OR EnvGroup = '';`); err != nil {
		return fmt.Errorf("failed to backfill EnvGroup on environment tests table: %v", err)
	}
	if _, err := m.db.Exec(`UPDATE db_test_cases SET EnvGroup = 'Legacy' WHERE EnvGroup IS NULL OR EnvGroup = '';`); err != nil {
		return fmt.Errorf("failed to backfill EnvGroup on test cases table: %v", err)
	}
	if _, err := m.db.Exec(`
		WITH env_groups AS (
			SELECT CommitID, EnvName, MIN(EnvGroup) AS EnvGroup
			FROM db_environment_tests
			WHERE EnvGroup IS NOT NULL AND EnvGroup <> ''
			GROUP BY CommitID, EnvName
			HAVING COUNT(DISTINCT EnvGroup) = 1
		)
		DELETE FROM db_test_cases
		WHERE (EnvGroup IS NULL OR EnvGroup = '' OR EnvGroup = 'Legacy')
			AND EXISTS (
				SELECT 1
				FROM env_groups eg
				WHERE eg.CommitID = db_test_cases.CommitId
					AND eg.EnvName = db_test_cases.EnvName
					AND EXISTS (
						SELECT 1
						FROM db_test_cases existing
						WHERE existing.CommitId = db_test_cases.CommitId
							AND existing.EnvName = db_test_cases.EnvName
							AND existing.TestName = db_test_cases.TestName
							AND existing.EnvGroup = eg.EnvGroup
					)
			);
	`); err != nil {
		return fmt.Errorf("failed to remove duplicate test cases during EnvGroup backfill: %v", err)
	}
	if _, err := m.db.Exec(`
		UPDATE db_test_cases
		SET EnvGroup = (
			SELECT env.EnvGroup
			FROM db_environment_tests env
			WHERE env.CommitID = db_test_cases.CommitId
				AND env.EnvName = db_test_cases.EnvName
				AND env.EnvGroup IS NOT NULL
				AND env.EnvGroup <> ''
			GROUP BY env.CommitID, env.EnvName
			HAVING COUNT(DISTINCT env.EnvGroup) = 1
		)
		WHERE (EnvGroup IS NULL OR EnvGroup = '' OR EnvGroup = 'Legacy')
			AND EXISTS (
				SELECT 1
				FROM db_environment_tests env
				WHERE env.CommitID = db_test_cases.CommitId
					AND env.EnvName = db_test_cases.EnvName
					AND env.EnvGroup IS NOT NULL
					AND env.EnvGroup <> ''
				GROUP BY env.CommitID, env.EnvName
				HAVING COUNT(DISTINCT env.EnvGroup) = 1
			);
	`); err != nil {
		return fmt.Errorf("failed to backfill EnvGroup on test cases table from environment tests: %v", err)
	}
	if err := ensureSQLitePrimaryKey(m.db, "db_environment_tests", []string{"CommitID", "EnvName", "EnvGroup"}, createEnvironmentTestsTableSQL, sqliteEnvTestsCopySQL); err != nil {
		return fmt.Errorf("failed to update environment tests primary key: %v", err)
	}
	if err := ensureSQLitePrimaryKey(m.db, "db_test_cases", []string{"CommitId", "EnvName", "EnvGroup", "TestName"}, createTestCasesTableSQL, sqliteTestCasesCopySQL); err != nil {
		return fmt.Errorf("failed to update test cases primary key: %v", err)
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

type sqliteColumnInfo struct {
	Name string `db:"name"`
	PK   int    `db:"pk"`
}

const sqliteEnvTestsCopySQL = `
	INSERT INTO %s (CommitID, EnvName, EnvGroup, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration, GopoghVersion, ArtifactPath)
	SELECT CommitID, EnvName, COALESCE(NULLIF(EnvGroup, ''), 'Legacy'), GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration, GopoghVersion, COALESCE(ArtifactPath, '')
	FROM %s;
`

const sqliteTestCasesCopySQL = `
	INSERT INTO %s (PR, CommitId, EnvName, EnvGroup, TestName, Result, Duration, TestOrder, TestTime)
	SELECT PR, CommitId, EnvName, COALESCE(NULLIF(EnvGroup, ''), 'Legacy'), TestName, Result, Duration, TestOrder, TestTime
	FROM %s;
`

func ensureSQLitePrimaryKey(db *sqlx.DB, table string, expectedPK []string, createSQL, insertSQL string) error {
	pkMatches, err := sqlitePrimaryKeyMatches(db, table, expectedPK)
	if err != nil || pkMatches {
		return err
	}
	return rebuildSQLiteTable(db, table, createSQL, insertSQL)
}

func sqlitePrimaryKeyMatches(db *sqlx.DB, table string, expectedPK []string) (bool, error) {
	var columns []sqliteColumnInfo
	if err := db.Select(&columns, fmt.Sprintf("PRAGMA table_info(%s);", table)); err != nil {
		return false, err
	}
	if len(columns) == 0 {
		return true, nil
	}
	pkCols := make([]string, 0, len(expectedPK))
	pkIndex := make(map[int]string)
	for _, col := range columns {
		if col.PK > 0 {
			pkIndex[col.PK] = col.Name
		}
	}
	for i := 1; i <= len(pkIndex); i++ {
		pkCols = append(pkCols, pkIndex[i])
	}
	if len(pkCols) != len(expectedPK) {
		return false, nil
	}
	for i := range expectedPK {
		if !strings.EqualFold(pkCols[i], expectedPK[i]) {
			return false, nil
		}
	}
	return true, nil
}

func rebuildSQLiteTable(db *sqlx.DB, table, createSQL, insertSQL string) error {
	newTable := table + "_new"
	createSQL = strings.Replace(createSQL, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s", table), fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s", newTable), 1)
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.Exec(createSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf(insertSQL, newTable, table)); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE %s;", table)); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", newTable, table)); err != nil {
		return err
	}
	return tx.Commit()
}

// GetEnvironmentTestsAndTestCases writes the database tables to a map with the keys environmentTests and testCases
// This is not yet supported for sqlite
func (m *sqlite) GetEnvironmentTestsAndTestCases() (map[string]interface{}, error) {
	return nil, nil
}

// GetEnvCharts writes the overall environment charts to a map with the keys recentFlakePercentTable, flakeRateByWeek, flakeRateByDay, and countsAndDurations
// This is not yet supported for sqlite
func (m *sqlite) GetEnvCharts(_ string, _ string, _ int) (map[string]interface{}, error) {
	return nil, nil
}

// GetTestCharts writes the individual test chart data to a map with the keys flakeByDay and flakeByWeek
// This is not yet supported for sqlite
func (m *sqlite) GetTestCharts(_ string, _ string, _ string) (map[string]interface{}, error) {
	return nil, nil
}

// GetOverview writes the overview charts to a map with the keys summaryAvgFail and summaryTable
// This is not yet supported for sqlite
func (m *sqlite) GetOverview(int) (map[string]interface{}, error) {
	return nil, nil
}
