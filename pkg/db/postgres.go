package db

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Blank import used for registering postgres driver as a database driver
	"github.com/medyagh/gopogh/pkg/models"
)

var pgEnvTableSchema = `
	CREATE TABLE IF NOT EXISTS db_environment_tests (
		CommitID TEXT,
		EnvName TEXT,
		GopoghTime TIMESTAMP,
		TestTime TIMESTAMP,
		NumberOfFail INTEGER,
		NumberOfPass INTEGER,
		NumberOfSkip INTEGER,
		TotalDuration FLOAT,
		PRIMARY KEY (CommitID, EnvName)
	);
`
var pgTestCasesTableSchema = `
	CREATE TABLE IF NOT EXISTS db_test_cases (
		PR TEXT,
		CommitID TEXT,
		EnvName TEXT,
		TestName TEXT,
		Result TEXT,
		TestTime TIMESTAMP,
		Duration FLOAT,
		PRIMARY KEY (CommitID, EnvName, TestName)
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
		INSERT INTO db_test_cases (PR, CommitId, EnvName, TestName, Result, TestTime, Duration)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (CommitId, EnvName, TestName)
		DO UPDATE SET (PR, Result, TestTime, Duration) = (EXCLUDED.PR, EXCLUDED.Result, EXCLUDED.TestTime, EXCLUDED.Duration)
	`
	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		return fmt.Errorf("failed to prepare SQL insert statement: %v", err)
	}
	defer stmt.Close()

	for _, r := range dbRows {
		_, err := stmt.Exec(r.PR, r.CommitID, r.EnvName, r.TestName, r.Result, r.TestTime, r.Duration)
		if err != nil {
			return fmt.Errorf("failed to execute SQL insert: %v", err)
		}
	}

	sqlInsert = `
		INSERT INTO db_environment_tests (CommitID, EnvName, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (CommitId, EnvName)
		DO UPDATE SET (GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration) = (EXCLUDED.GopoghTime, EXCLUDED.TestTime, EXCLUDED.NumberOfFail, EXCLUDED.NumberOfPass, EXCLUDED.NumberOfSkip, EXCLUDED.TotalDuration)
		`
	_, err = tx.Exec(sqlInsert, commitRow.CommitID, commitRow.EnvName, commitRow.GopoghTime, commitRow.TestTime, commitRow.NumberOfFail, commitRow.NumberOfPass, commitRow.NumberOfSkip, commitRow.TotalDuration)
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
	path := fmt.Sprintf("host=%s %s", cfg.host, cfg.path)
	database, err := sqlx.Connect("postgres", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}
	m := &Postgres{
		db:   database,
		path: path,
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

// PrintEnvironmentTestsAndTestCases writes the environment tests and test cases tables to an HTTP response in a combined page
func (m *Postgres) PrintEnvironmentTestsAndTestCases(w http.ResponseWriter, _ *http.Request) {
	var environmentTests []models.DBEnvironmentTest
	var testCases []models.DBTestCase

	err := m.db.Select(&environmentTests, "SELECT * FROM db_environment_tests")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to execute SQL query for environment tests: %v", err), http.StatusInternalServerError)
		return
	}

	err = m.db.Select(&testCases, "SELECT * FROM db_test_cases")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to execute SQL query for test cases: %v", err), http.StatusInternalServerError)
		return
	}

	// Write the response header to be html
	w.Header().Set("Content-Type", "text/html")

	// Write the HTML page structure
	fmt.Fprintf(w, "<html><head><title>Environment Tests and Test Cases</title></head><body>")

	// Environment tests table
	fmt.Fprintf(w, "<h1>Environment Tests</h1><table>")
	fmt.Fprintf(w, "<thead><tr><th>CommitID</th><th>EnvName</th><th>GopoghTime</th><th>TestTime</th><th>NumberOfFail</th><th>NumberOfPass</th><th>NumberOfSkip</th><th>TotalDuration</th></tr></thead>")
	for _, row := range environmentTests {
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%f</td></tr>",
			row.CommitID, row.EnvName, row.GopoghTime, row.TestTime, row.NumberOfFail, row.NumberOfPass, row.NumberOfSkip, row.TotalDuration)
	}
	fmt.Fprintf(w, "</table>")

	// Test cases table
	fmt.Fprintf(w, "<h1>Test Cases</h1><table>")
	fmt.Fprintf(w, "<thead><tr><th>PR</th><th>CommitID</th><th>EnvName</th><th>TestName</th><th>Result</th><th>TestTime</th><th>Duration</th></tr></thead>")
	for _, row := range testCases {
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%f</td></tr>",
			row.PR, row.CommitID, row.EnvName, row.TestName, row.Result, row.TestTime, row.Duration)
	}
	fmt.Fprintf(w, "</table>")

	// Close the HTML page structure
	fmt.Fprintf(w, "</body></html>")
}

// PrintBasicFlake writes the a basic flake rate table to an HTTP response
func (m *Postgres) PrintBasicFlake(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	env := queryValues.Get("env")
	if env == "" {
		env = "KVM Linux"
	}

	// Number of days to use to look for "flaky-est" tests.
	const dateRange = 15

	// This query first makes a temp table containing the $1 (30) most recent dates
	// Then it computes the recentCutoff and prevCutoff (15th most recent and 30th most recent dates)
	// Then, filtering out the skips and filtering for the correct env we calculate the flake rate and the flake rate growth
	// for the 15 most recent days and the 15 days following that
	sqlQuer := `
	WITH dates AS (
		SELECT DISTINCT DATE_TRUNC('day', TestTime) AS Date
		FROM db_test_cases
		ORDER BY Date DESC
		LIMIT $1
	), recentCutoff AS (
		SELECT Date 
		FROM dates 
		OFFSET $2
		LIMIT 1
	), prevCutoff AS (
		SELECT Date
		FROM dates
		OFFSET $3
		LIMIT 1
	)
	SELECT TestName,
	ROUND(COALESCE(AVG(CASE WHEN TestTime > (SELECT Date From prevCutoff) THEN CASE WHEN Result = 'fail' THEN 1 ELSE 0 END END) * 100, 0), 2) AS RecentFlakePercentage,
	ROUND(100.0 * COALESCE((AVG(CASE WHEN TestTime > (SELECT Date From recentCutoff) THEN CASE WHEN Result = 'fail' THEN 1 ELSE 0 END END) - AVG(CASE WHEN TestTime <= (SELECT Date From recentCutoff) AND TestTime > (SELECT Date From prevCutoff) THEN CASE WHEN Result = 'fail' THEN 1 ELSE 0 END END)) / NULLIF(AVG(CASE WHEN TestTime <= (SELECT Date From recentCutoff) THEN 1 ELSE 0 END), 0), 0), 2) AS GrowthRate
	FROM db_test_cases
	WHERE Result != 'skip' AND EnvName = $4
	GROUP BY TestName
	ORDER BY RecentFlakePercentage DESC;
	`
	var flakeRates []models.DBFlakeRow
	err := m.db.Select(&flakeRates, sqlQuer, 2*dateRange, dateRange-1, 2*dateRange-1, env)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to execute SQL query for flake chart: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"recentFlakePercentTable": flakeRates,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, "Failed to write JSON data", http.StatusInternalServerError)
		return
	}
}
