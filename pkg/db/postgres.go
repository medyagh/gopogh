package db

import (
	"fmt"
	"log"
	"strings"
	"time"

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

// GetEnvironmentTestsAndTestCases writes the database tables to a map with the keys environmentTests and testCases
func (m *Postgres) GetEnvironmentTestsAndTestCases() (map[string]interface{}, error) {
	start := time.Now()

	var environmentTests []models.DBEnvironmentTest
	var testCases []models.DBTestCase

	err := m.db.Select(&environmentTests, "SELECT * FROM db_environment_tests ORDER BY TestTime DESC LIMIT 100")
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for environment tests: %v", err)
	}

	err = m.db.Select(&testCases, "SELECT * FROM db_test_cases ORDER BY TestTime DESC LIMIT 100")
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for test cases: %v", err)

	}
	data := map[string]interface{}{
		"environmentTests": environmentTests,
		"testCases":        testCases,
	}
	log.Printf("\nduration metric: took %f seconds to gather all table data since start of handler\n\n", time.Since(start).Seconds())
	return data, nil
}

func (m *Postgres) createMaterializedView(env string, viewName string) error {
	createView := fmt.Sprintf(`
	CREATE MATERIALIZED VIEW IF NOT EXISTS %s AS 
		SELECT * FROM db_test_cases
		WHERE Result != 'skip' AND EnvName = '%s' AND TestTime >= NOW() - INTERVAL '90 days'
	`, viewName, env)

	_, err := m.db.Exec(createView)
	return err
}

// GetTestCharts writes the individual test chart data to a map with the keys flakeByDay and flakeByWeek
func (m *Postgres) GetTestCharts(env string, test string) (map[string]interface{}, error) {
	start := time.Now()

	var validEnvs []string
	err := m.db.Select(&validEnvs, "SELECT DISTINCT EnvName FROM db_environment_tests")
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for list of valid environments: %v", err)
	}
	isValidEnv := false
	for _, e := range validEnvs {
		if env == e {
			isValidEnv = true
		}
	}
	if !isValidEnv {
		return nil, fmt.Errorf("invalid environment. Not found in database: %v", err)
	}

	viewName := fmt.Sprintf("\"lastn_data_%s\"", env)
	err = m.createMaterializedView(env, viewName)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for view creation: %v", err)
	}

	log.Printf("\nduration metric: took %f seconds to execute SQL query for refreshing materialized view since start of handler", time.Since(start).Seconds())

	// Groups the datetimes together by date, calculating flake percentage and aggregating the individual results/durations for each date
	sqlQuery := fmt.Sprintf(`
	SELECT
	DATE_TRUNC('day', TestTime) AS StartOfDate,
	AVG(Duration) AS AvgDuration,
	ROUND(COALESCE(AVG(CASE WHEN Result = 'fail' THEN 1 ELSE 0 END) * 100, 0), 2) AS FlakePercentage,
	STRING_AGG(CommitID || ': ' || Result || ': ' || Duration, ', ') AS CommitResultsAndDurations
	FROM %s 
	WHERE TestName = $1
	GROUP BY StartOfDate
	ORDER BY StartOfDate DESC
	`, viewName)

	var flakeByDay []models.DBTestRateAndDuration
	err = m.db.Select(&flakeByDay, sqlQuery, test)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for flake rate and duration by day chart: %v", err)
	}

	log.Printf("\nduration metric: took %f seconds to execute SQL query for flake rate and duration by day chart since start of handler", time.Since(start).Seconds())

	// Groups the datetimes together by week, calculating flake percentage and aggregating the individual results/durations for each date
	sqlQuery = fmt.Sprintf(`
	SELECT
	DATE_TRUNC('week', TestTime) AS StartOfDate,
	AVG(Duration) AS AvgDuration,
	ROUND(COALESCE(AVG(CASE WHEN Result = 'fail' THEN 1 ELSE 0 END) * 100, 0), 2) AS FlakePercentage,
	STRING_AGG(CommitID || ': ' || Result || ': ' || Duration, ', ') AS CommitResultsAndDurations
	FROM %s 
	WHERE TestName = $1
	GROUP BY StartOfDate
	ORDER BY StartOfDate DESC
	`, viewName)
	var flakeByWeek []models.DBTestRateAndDuration
	err = m.db.Select(&flakeByWeek, sqlQuery, test)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for flake rate and duration by week chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for flake rate and duration by week chart since start of handler", time.Since(start).Seconds())

	data := map[string]interface{}{
		"flakeByDay":  flakeByDay,
		"flakeByWeek": flakeByWeek,
	}
	log.Printf("\nduration metric: took %f seconds to gather individual test chart data since start of handler\n\n", time.Since(start).Seconds())
	return data, nil
}

// GetEnvCharts writes the overall environment charts to a map with the keys recentFlakePercentTable, flakeRateByWeek, flakeRateByDay, and countsAndDurations
func (m *Postgres) GetEnvCharts(env string, testsInTop int) (map[string]interface{}, error) {
	start := time.Now()

	var validEnvs []string
	err := m.db.Select(&validEnvs, "SELECT DISTINCT EnvName FROM db_environment_tests")
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for list of valid environments: %v", err)
	}
	isValidEnv := false
	for _, e := range validEnvs {
		if env == e {
			isValidEnv = true
		}
	}
	if !isValidEnv {
		return nil, fmt.Errorf("invalid environment. Not found in database: %v", err)
	}

	viewName := fmt.Sprintf("\"lastn_data_%s\"", env)
	err = m.createMaterializedView(env, viewName)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for view creation: %v", err)
	}

	log.Printf("\nduration metric: took %f seconds to execute SQL query for refreshing materialized view since start of handler", time.Since(start).Seconds())

	// Number of days to use to look for "flaky-est" tests.
	const dateRange = 15

	// This query first makes a temp table containing the $1 (30) most recent dates
	// Then it computes the recentCutoff and prevCutoff (15th most recent and 30th most recent dates)
	// Then we calculate the flake rate and the flake rate growth
	// for the 15 most recent days and the 15 days following that
	sqlQuer := fmt.Sprintf(`
	WITH dates AS (
		SELECT DISTINCT DATE_TRUNC('day', TestTime) AS Date
		FROM %s
		ORDER BY Date DESC
		LIMIT $1
	), recentCutoff AS (
		SELECT Date 
		FROM dates 
		ORDER BY Date DESC
		OFFSET $2
		LIMIT 1
	), prevCutoff AS (
		SELECT Date
		FROM dates
		ORDER BY Date DESC
		OFFSET $3
		LIMIT 1
	), temp AS (
	SELECT TestName,
	ROUND(COALESCE(AVG(CASE WHEN TestTime > (SELECT Date FROM recentCutoff) THEN CASE WHEN Result = 'fail' THEN 1 ELSE 0 END END) * 100, 0), 2) AS RecentFlakePercentage,
	ROUND(COALESCE(AVG(CASE WHEN TestTime <= (SELECT Date FROM recentCutoff) AND TestTime > (SELECT Date FROM prevCutoff) THEN CASE WHEN Result = 'fail' THEN 1 ELSE 0 END END) * 100, 0), 2) AS PrevFlakePercentage
	FROM %s
	GROUP BY TestName
	ORDER BY RecentFlakePercentage DESC
	)
	SELECT TestName, RecentFlakePercentage, RecentFlakePercentage - PrevFlakePercentage AS GrowthRate
	FROM temp
	ORDER BY RecentFlakePercentage DESC;
	`, viewName, viewName)
	var flakeRates []models.DBFlakeRow
	err = m.db.Select(&flakeRates, sqlQuer, 2*dateRange, dateRange-1, 2*dateRange-1)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for flake table: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for flake table since start of handler", time.Since(start).Seconds())

	var topTestNames []string
	for _, row := range flakeRates {
		topTestNames = append(topTestNames, row.TestName)
		if len(topTestNames) >= testsInTop {
			break
		}
	}

	// Gets the data on just the top ten previously calculated and aggregates flake rates and results per date
	sqlQuer = fmt.Sprintf(`
	WITH lastn_data_top AS (
		SELECT *
		FROM %s
		WHERE TestName IN ('%s')
	)
	SELECT TestName, 
	DATE_TRUNC('day', TestTime) AS StartOfDate,
	COALESCE(AVG(CASE WHEN Result = 'fail' THEN 1 ELSE 0 END) * 100, 0) AS FlakePercentage,
	STRING_AGG(CommitID || ': ' || Result, ', ') AS CommitResults
	FROM lastn_data_top
	GROUP BY TestName, StartOfDate
	ORDER BY StartOfDate DESC
	`, viewName,
		strings.Join(topTestNames, "', '"))
	var flakeRateByDay []models.DBFlakeBy
	err = m.db.Select(&flakeRateByDay, sqlQuer)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for by day flake chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for day flake chart since start of handler", time.Since(start).Seconds())

	// Filters to get the top flakiest in the past week, calculating flake rate per week for those tests
	sqlQuer = fmt.Sprintf(`
	WITH recent_week AS (
		SELECT MAX (DATE_TRUNC('week', TestTime)) AS weekCutoff
		FROM %s
	),
	recent_week_data AS (
		SELECT * 
		FROM %s 
		WHERE TestTime >= (SELECT weekCutoff FROM recent_week)
	),
	top_flakiest AS (
		SELECT TestName, COALESCE(AVG(CASE WHEN Result = 'fail' THEN 1 ELSE 0 END) * 100, 0) AS RecentFlakePercentage
		FROM recent_week_data
		GROUP BY TestName
		ORDER BY RecentFlakePercentage DESC
		LIMIT $1
	),
	top_flakiest_data AS (
		SELECT * FROM %s 
		WHERE TestName IN (SELECT TestName FROM top_flakiest)
	)
	SELECT TestName,
	DATE_TRUNC('week', TestTime) AS StartOfDate,
	ROUND(COALESCE(AVG(CASE WHEN Result = 'fail' THEN 1 ELSE 0 END) * 100, 0), 2) AS FlakePercentage,
	STRING_AGG(CommitID || ': ' || Result, ', ') AS CommitResults
	FROM top_flakiest_data
	GROUP BY TestName, StartOfDate
	ORDER BY StartOfDate DESC;
	`, viewName, viewName, viewName)
	var flakeRateByWeek []models.DBFlakeBy
	err = m.db.Select(&flakeRateByWeek, sqlQuer, testsInTop)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for by week flake chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for flake by week chart since start of handler", time.Since(start).Seconds())

	// Filters out data prior to 90 days and with the incorrect environment
	// Then calculates for each date aggregates the duration and number of tests, calculating the average for both
	sqlQuer = `
	WITH lastn_env_data AS (
		SELECT *
		FROM db_environment_tests
		WHERE EnvName = $1 AND TestTime >= NOW() - INTERVAL '90 days'
	)
	SELECT
	DATE_TRUNC('day', TestTime) AS StartOfDate,
	AVG(NumberOfPass + NumberOfFail) AS TestCount,
	AVG(TotalDuration) AS Duration,
	STRING_AGG(CommitID || ': ' || (NumberOfPass + NumberOfFail), ', ') AS CommitCounts,
	STRING_AGG(CommitID || ': ' || TotalDuration, ', ') AS CommitDurations
	FROM lastn_env_data 
	GROUP BY StartOfDate
	ORDER BY StartOfDate DESC
	`
	var countsAndDurations []models.DBEnvDuration
	err = m.db.Select(&countsAndDurations, sqlQuer, env)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for environment test count and duration chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for env duration chart since start of handler", time.Since(start).Seconds())

	data := map[string]interface{}{
		"recentFlakePercentTable": flakeRates,
		"flakeRateByWeek":         flakeRateByWeek,
		"flakeRateByDay":          flakeRateByDay,
		"countsAndDurations":      countsAndDurations,
	}
	log.Printf("\nduration metric: took %f seconds to gather env chart data since start of handler\n\n", time.Since(start).Seconds())
	return data, nil
}

// GetOverview writes the overview charts to a map with the keys summaryAvgFail and summaryTable
func (m *Postgres) GetOverview() (map[string]interface{}, error) {
	start := time.Now()
	// Filters out old data and calculates the average number of failures and average duration per day per environment
	sqlQuery := `
	SELECT DATE_TRUNC('day', TestTime) AS StartOfDate, EnvName, AVG(NumberOfFail) AS AvgFailedTests, AVG(TotalDuration) AS AvgDuration
	FROM db_environment_tests
	WHERE TestTime >= NOW() - INTERVAL '90 days'
	GROUP BY StartOfDate, EnvName
	ORDER BY StartOfDate, EnvName;
	`

	var summaryAvgFail []models.DBSummaryAvgFail
	err := m.db.Select(&summaryAvgFail, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for summary chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for summary duration and failure charts since start of handler", time.Since(start).Seconds())

	// Number of days to use to look for "flaky-est" envs.
	const dateRange = 15

	// Filters out data from prior to 90 days
	// Then computes average number of fails for each environment for each time frame
	// Then calculates the change in the average number of fails between the time frames
	sqlQuery = `
	WITH data AS (
		SELECT * 
		FROM db_environment_tests 
		WHERE TestTime >= NOW() - INTERVAL '90 days'
	), dates AS (
		SELECT DISTINCT DATE_TRUNC('day', TestTime) AS Date
		FROM data
		ORDER BY Date DESC
		LIMIT $1
	), recentCutoff AS (
		SELECT Date 
		FROM dates 
		ORDER BY Date DESC
		OFFSET $2
		LIMIT 1
	), prevCutoff AS (
		SELECT Date
		FROM dates
		ORDER BY Date DESC
		OFFSET $3
		LIMIT 1
	), temp AS (
	SELECT EnvName,
	ROUND(COALESCE(AVG(CASE WHEN TestTime > (SELECT Date FROM recentCutoff) THEN NumberOfFail END), 0), 2) AS RecentNumberOfFail,
	ROUND(COALESCE(AVG(CASE WHEN TestTime <= (SELECT Date FROM recentCutoff) AND TestTime > (SELECT Date FROM prevCutoff) THEN NumberOfFail END), 0), 2) AS PrevNumberOfFail
	FROM data
	GROUP BY EnvName
	ORDER BY RecentNumberOfFail DESC
	)
	SELECT EnvName, RecentNumberOfFail, RecentNumberOfFail - PrevNumberOfFail AS Growth
	FROM temp
	ORDER BY RecentNumberOfFail DESC;
	`
	var summaryTable []models.DBSummaryTable
	err = m.db.Select(&summaryTable, sqlQuery, 2*dateRange, dateRange-1, 2*dateRange-1)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for flake table: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for summary failure change table since start of handler", time.Since(start).Seconds())

	data := map[string]interface{}{
		"summaryAvgFail": summaryAvgFail,
		"summaryTable":   summaryTable,
	}
	log.Printf("\nduration metric: took %f seconds to gather summary data since start of handler\n\n", time.Since(start).Seconds())
	return data, nil
}
