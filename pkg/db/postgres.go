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
		EnvGroup TEXT NOT NULL DEFAULT 'Legacy',
		GopoghTime TIMESTAMP,
		TestTime TIMESTAMP,
		NumberOfFail INTEGER,
		NumberOfPass INTEGER,
		NumberOfSkip INTEGER,
		TotalDuration FLOAT,
		ArtifactPath TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (CommitID, EnvName, EnvGroup)
	);
`
var pgTestCasesTableSchema = `
	CREATE TABLE IF NOT EXISTS db_test_cases (
		PR TEXT,
		CommitID TEXT,
		EnvName TEXT,
		EnvGroup TEXT NOT NULL DEFAULT 'Legacy',
		TestName TEXT,
		Result TEXT,
		TestTime TIMESTAMP,
		Duration FLOAT,
		PRIMARY KEY (CommitID, EnvName, EnvGroup, TestName)
	);
`

// Postgres is a Postgres database database struct instance
type Postgres struct {
	db   *sqlx.DB
	path string
}

// Set adds/updates rows to the database
func (m *Postgres) Set(commitRow models.DBEnvironmentTest, dbRows []models.DBTestCase) error {
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

	sqlInsert := `
		INSERT INTO db_test_cases (PR, CommitId, EnvName, EnvGroup, TestName, Result, TestTime, Duration)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (CommitId, EnvName, EnvGroup, TestName)
		DO UPDATE SET (PR, Result, TestTime, Duration) = (EXCLUDED.PR, EXCLUDED.Result, EXCLUDED.TestTime, EXCLUDED.Duration)
	`
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
		_, err := stmt.Exec(r.PR, r.CommitID, r.EnvName, rowEnvGroup, r.TestName, r.Result, r.TestTime, r.Duration)
		if err != nil {
			return fmt.Errorf("failed to execute SQL insert: %v", err)
		}
	}

	sqlInsert = `
		INSERT INTO db_environment_tests (CommitID, EnvName, EnvGroup, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration, ArtifactPath) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (CommitId, EnvName, EnvGroup)
		DO UPDATE SET (EnvGroup, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration, ArtifactPath) = (EXCLUDED.EnvGroup, EXCLUDED.GopoghTime, EXCLUDED.TestTime, EXCLUDED.NumberOfFail, EXCLUDED.NumberOfPass, EXCLUDED.NumberOfSkip, EXCLUDED.TotalDuration, EXCLUDED.ArtifactPath)
		`
	_, err = tx.Exec(sqlInsert, commitRow.CommitID, commitRow.EnvName, envGroup, commitRow.GopoghTime, commitRow.TestTime, commitRow.NumberOfFail, commitRow.NumberOfPass, commitRow.NumberOfSkip, commitRow.TotalDuration, commitRow.ArtifactPath)
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
	start := time.Now()
	defer func() {
		log.Printf("\nduration metric: took %f seconds to initialize Postgres tables\n", time.Since(start).Seconds())
	}()

	if _, err := m.db.Exec(pgEnvTableSchema); err != nil {
		return fmt.Errorf("failed to initialize environment tests table: %v", err)
	}
	if _, err := m.db.Exec(pgTestCasesTableSchema); err != nil {
		return fmt.Errorf("failed to initialize test cases table: %v", err)
	}
	if _, err := m.db.Exec(`ALTER TABLE db_environment_tests ADD COLUMN IF NOT EXISTS EnvGroup TEXT NOT NULL DEFAULT 'Legacy';`); err != nil {
		return fmt.Errorf("failed to ensure EnvGroup on environment tests table: %v", err)
	}
	if _, err := m.db.Exec(`ALTER TABLE db_environment_tests ADD COLUMN IF NOT EXISTS ArtifactPath TEXT NOT NULL DEFAULT '';`); err != nil {
		return fmt.Errorf("failed to ensure ArtifactPath on environment tests table: %v", err)
	}
	if _, err := m.db.Exec(`ALTER TABLE db_test_cases ADD COLUMN IF NOT EXISTS EnvGroup TEXT NOT NULL DEFAULT 'Legacy';`); err != nil {
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
		DELETE FROM db_test_cases tc
		USING env_groups eg
		WHERE tc.CommitID = eg.CommitID
			AND tc.EnvName = eg.EnvName
			AND (tc.EnvGroup IS NULL OR tc.EnvGroup = '' OR tc.EnvGroup = 'Legacy')
			AND EXISTS (
				SELECT 1 FROM db_test_cases existing
				WHERE existing.CommitID = tc.CommitID
					AND existing.EnvName = tc.EnvName
					AND existing.TestName = tc.TestName
					AND existing.EnvGroup = eg.EnvGroup
			);
	`); err != nil {
		return fmt.Errorf("failed to remove duplicate test cases during EnvGroup backfill: %v", err)
	}
	if _, err := m.db.Exec(`
		WITH env_groups AS (
			SELECT CommitID, EnvName, MIN(EnvGroup) AS EnvGroup
			FROM db_environment_tests
			WHERE EnvGroup IS NOT NULL AND EnvGroup <> ''
			GROUP BY CommitID, EnvName
			HAVING COUNT(DISTINCT EnvGroup) = 1
		)
		UPDATE db_test_cases tc
		SET EnvGroup = eg.EnvGroup
		FROM env_groups eg
		WHERE tc.CommitID = eg.CommitID
			AND tc.EnvName = eg.EnvName
			AND (tc.EnvGroup IS NULL OR tc.EnvGroup = '' OR tc.EnvGroup = 'Legacy');
	`); err != nil {
		return fmt.Errorf("failed to backfill EnvGroup on test cases table from environment tests: %v", err)
	}
	if _, err := m.db.Exec(`ALTER TABLE db_environment_tests DROP CONSTRAINT IF EXISTS db_environment_tests_pkey;`); err != nil {
		return fmt.Errorf("failed to drop environment tests primary key: %v", err)
	}
	if _, err := m.db.Exec(`ALTER TABLE db_environment_tests ADD PRIMARY KEY (CommitID, EnvName, EnvGroup);`); err != nil {
		return fmt.Errorf("failed to add environment tests primary key: %v", err)
	}
	if _, err := m.db.Exec(`ALTER TABLE db_test_cases DROP CONSTRAINT IF EXISTS db_test_cases_pkey;`); err != nil {
		return fmt.Errorf("failed to drop test cases primary key: %v", err)
	}
	if _, err := m.db.Exec(`ALTER TABLE db_test_cases ADD PRIMARY KEY (CommitID, EnvName, EnvGroup, TestName);`); err != nil {
		return fmt.Errorf("failed to add test cases primary key: %v", err)
	}
	return nil
}

// GetEnvironmentTestsAndTestCases writes the database tables to a map with the keys environmentTests and testCases
func (m *Postgres) GetEnvironmentTestsAndTestCases() (map[string]interface{}, error) {
	start := time.Now()

	var environmentTests []models.DBEnvironmentTest
	var testCases []models.DBTestCase

	err := m.db.Select(&environmentTests, `
		SELECT CommitID, EnvName, EnvGroup, GopoghTime, TestTime, NumberOfFail, NumberOfPass, NumberOfSkip, TotalDuration, ArtifactPath
		FROM db_environment_tests
		ORDER BY TestTime DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for environment tests: %v", err)
	}

	err = m.db.Select(&testCases, `
		SELECT PR, CommitID, EnvName, EnvGroup, TestName, Result, TestTime, Duration
		FROM db_test_cases
		ORDER BY TestTime DESC
		LIMIT 100
	`)
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

func sanitizeIdentifier(value string) string {
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	result := b.String()
	if result == "" {
		return "unknown"
	}
	return result
}

func escapeSQLLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func materializedViewName(envName, envGroup string) string {
	base := fmt.Sprintf("lastn_data_%s_%s", sanitizeIdentifier(envGroup), sanitizeIdentifier(envName))
	return fmt.Sprintf("\"%s\"", base)
}

func (m *Postgres) resolveEnvGroup(envName, envGroup string) (string, error) {
	envName = strings.TrimSpace(envName)
	envGroup = strings.TrimSpace(envGroup)
	if envName == "" {
		return "", fmt.Errorf("missing environment name")
	}
	if envGroup != "" {
		var exists bool
		if err := m.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM db_environment_tests WHERE EnvName = $1 AND EnvGroup = $2)", envName, envGroup); err != nil {
			return "", fmt.Errorf("failed to validate environment: %v", err)
		}
		if !exists {
			return "", fmt.Errorf("invalid environment group for %s: %s", envName, envGroup)
		}
		return envGroup, nil
	}
	var groups []string
	if err := m.db.Select(&groups, "SELECT DISTINCT EnvGroup FROM db_environment_tests WHERE EnvName = $1", envName); err != nil {
		return "", fmt.Errorf("failed to execute SQL query for list of valid environment groups: %v", err)
	}
	if len(groups) == 0 {
		return "", fmt.Errorf("invalid environment. Not found in database: %v", envName)
	}
	if len(groups) > 1 {
		return "", fmt.Errorf("environment name %q exists in multiple env groups; specify env_group", envName)
	}
	group := strings.TrimSpace(groups[0])
	if group == "" {
		group = "Legacy"
	}
	return group, nil
}

func (m *Postgres) createMaterializedView(envName, envGroup, viewName string) error {
	envName = escapeSQLLiteral(envName)
	envGroup = escapeSQLLiteral(envGroup)
	createView := fmt.Sprintf(`
	CREATE MATERIALIZED VIEW IF NOT EXISTS %s AS 
		SELECT CommitID, EnvName, EnvGroup, TestName, Result, Duration, TestTime FROM db_test_cases
		WHERE Result != 'skip' AND EnvName = '%s' AND EnvGroup = '%s' AND TestTime >= NOW() - INTERVAL '90 days'
	`, viewName, envName, envGroup)

	if _, err := m.db.Exec(createView); err != nil {
		return err
	}
	// if we add a new test environment the service account will be the owner of the newly created materalized view above
	// but we require postgres to be the owner for the cron that refreshes the materialized view to run
	alterOwner := fmt.Sprintf("ALTER MATERIALIZED VIEW %s OWNER TO postgres;", viewName)
	if _, err := m.db.Exec(alterOwner); err != nil {
		return err
	}
	return nil
}

// GetTestCharts writes the individual test chart data to a map with the keys flakeByDay and flakeByWeek
func (m *Postgres) GetTestCharts(envName string, envGroup string, test string) (map[string]interface{}, error) {
	start := time.Now()

	resolvedEnvGroup, err := m.resolveEnvGroup(envName, envGroup)
	if err != nil {
		return nil, err
	}

	viewName := materializedViewName(envName, resolvedEnvGroup)
	err = m.createMaterializedView(envName, resolvedEnvGroup, viewName)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for view creation: %v", err)
	}

	log.Printf("\nduration metric: took %f seconds to execute SQL query for refreshing materialized view since start of handler", time.Since(start).Seconds())

	// Groups the datetimes together by date, calculating flake percentage and aggregating the individual results/durations for each date
	sqlQuery := fmt.Sprintf(`
	SELECT
	DATE_TRUNC('day', tc.TestTime) AS StartOfDate,
	AVG(tc.Duration) AS AvgDuration,
	ROUND(COALESCE(AVG(CASE WHEN tc.Result = 'fail' THEN 1 ELSE 0 END) * 100, 0), 2) AS FlakePercentage,
	STRING_AGG(tc.CommitID || ': ' || tc.Result || ': ' || tc.Duration || ': ' || COALESCE(env.ArtifactPath, ''), ', ') AS CommitResultsAndDurations
	FROM %s AS tc
	LEFT JOIN db_environment_tests env
	ON env.CommitID = tc.CommitID AND env.EnvName = tc.EnvName AND env.EnvGroup = tc.EnvGroup
	WHERE tc.TestName = $1
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
	DATE_TRUNC('week', tc.TestTime) AS StartOfDate,
	AVG(tc.Duration) AS AvgDuration,
	ROUND(COALESCE(AVG(CASE WHEN tc.Result = 'fail' THEN 1 ELSE 0 END) * 100, 0), 2) AS FlakePercentage,
	STRING_AGG(tc.CommitID || ': ' || tc.Result || ': ' || tc.Duration || ': ' || COALESCE(env.ArtifactPath, ''), ', ') AS CommitResultsAndDurations
	FROM %s AS tc
	LEFT JOIN db_environment_tests env
	ON env.CommitID = tc.CommitID AND env.EnvName = tc.EnvName AND env.EnvGroup = tc.EnvGroup
	WHERE tc.TestName = $1
	GROUP BY StartOfDate
	ORDER BY StartOfDate DESC
	`, viewName)
	var flakeByWeek []models.DBTestRateAndDuration
	err = m.db.Select(&flakeByWeek, sqlQuery, test)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for flake rate and duration by week chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for flake rate and duration by week chart since start of handler", time.Since(start).Seconds())

	// Groups the datetimes together by month, calculating flake percentage and aggregating the individual results/durations for each date
	sqlQuery = fmt.Sprintf(`
	SELECT
	DATE_TRUNC('month', tc.TestTime) AS StartOfDate,
	AVG(tc.Duration) AS AvgDuration,
	ROUND(COALESCE(AVG(CASE WHEN tc.Result = 'fail' THEN 1 ELSE 0 END) * 100, 0), 2) AS FlakePercentage,
	STRING_AGG(tc.CommitID || ': ' || tc.Result || ': ' || tc.Duration || ': ' || COALESCE(env.ArtifactPath, ''), ', ') AS CommitResultsAndDurations
	FROM %s AS tc
	LEFT JOIN db_environment_tests env
	ON env.CommitID = tc.CommitID AND env.EnvName = tc.EnvName AND env.EnvGroup = tc.EnvGroup
	WHERE tc.TestName = $1
	GROUP BY StartOfDate
	ORDER BY StartOfDate DESC
	`, viewName)
	var flakeByMonth []models.DBTestRateAndDuration
	err = m.db.Select(&flakeByMonth, sqlQuery, test)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for flake rate and duration by month chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for flake rate and duration by month chart since start of handler", time.Since(start).Seconds())

	data := map[string]interface{}{
		"flakeByDay":   flakeByDay,
		"flakeByWeek":  flakeByWeek,
		"flakeByMonth": flakeByMonth,
	}
	log.Printf("\nduration metric: took %f seconds to gather individual test chart data since start of handler\n\n", time.Since(start).Seconds())
	return data, nil
}

// GetEnvCharts writes the overall environment charts to a map with the keys recentFlakePercentTable, flakeRateByWeek, flakeRateByDay, and countsAndDurations
func (m *Postgres) GetEnvCharts(envName string, envGroup string, testsInTop int) (map[string]interface{}, error) {
	start := time.Now()

	resolvedEnvGroup, err := m.resolveEnvGroup(envName, envGroup)
	if err != nil {
		return nil, err
	}

	viewName := materializedViewName(envName, resolvedEnvGroup)
	err = m.createMaterializedView(envName, resolvedEnvGroup, viewName)
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
	SUM(CASE WHEN Result = 'fail' AND TestTime > (SELECT Date FROM recentCutoff) THEN 1 ELSE 0 END) As FailedTestNum,
	SUM(CASE WHEN TestTime > (SELECT Date FROM recentCutoff) THEN 1 ELSE 0 END) As TotalTestNum,
	ROUND(COALESCE(AVG(CASE WHEN TestTime > (SELECT Date FROM recentCutoff) THEN CASE WHEN Result = 'fail' THEN 1 ELSE 0 END END) * 100, 0), 2) AS RecentFlakePercentage,
	ROUND(COALESCE(AVG(CASE WHEN TestTime <= (SELECT Date FROM recentCutoff) AND TestTime > (SELECT Date FROM prevCutoff) THEN CASE WHEN Result = 'fail' THEN 1 ELSE 0 END END) * 100, 0), 2) AS PrevFlakePercentage
	FROM %s
	GROUP BY TestName
	ORDER BY RecentFlakePercentage DESC
	)
	SELECT TestName, RecentFlakePercentage, RecentFlakePercentage - PrevFlakePercentage AS GrowthRate, FailedTestNum, TotalTestNum
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
	SELECT tc.TestName, 
	DATE_TRUNC('day', tc.TestTime) AS StartOfDate,
	COALESCE(AVG(CASE WHEN tc.Result = 'fail' THEN 1 ELSE 0 END) * 100, 0) AS FlakePercentage,
	STRING_AGG(tc.CommitID || ': ' || tc.Result || ': ' || COALESCE(env.ArtifactPath, ''), ', ') AS CommitResults
	FROM lastn_data_top AS tc
	LEFT JOIN db_environment_tests env
	ON env.CommitID = tc.CommitID AND env.EnvName = tc.EnvName AND env.EnvGroup = tc.EnvGroup
	GROUP BY tc.TestName, StartOfDate
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
	SELECT tc.TestName,
	DATE_TRUNC('week', tc.TestTime) AS StartOfDate,
	ROUND(COALESCE(AVG(CASE WHEN tc.Result = 'fail' THEN 1 ELSE 0 END) * 100, 0), 2) AS FlakePercentage,
	STRING_AGG(tc.CommitID || ': ' || tc.Result || ': ' || COALESCE(env.ArtifactPath, ''), ', ') AS CommitResults
	FROM top_flakiest_data AS tc
	LEFT JOIN db_environment_tests env
	ON env.CommitID = tc.CommitID AND env.EnvName = tc.EnvName AND env.EnvGroup = tc.EnvGroup
	GROUP BY tc.TestName, StartOfDate
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
		WHERE EnvName = $1 AND EnvGroup = $2 AND TestTime >= NOW() - INTERVAL '90 days'
	)
	SELECT
	DATE_TRUNC('day', TestTime) AS StartOfDate,
	AVG(NumberOfPass + NumberOfFail) AS TestCount,
	AVG(TotalDuration) AS Duration,
	STRING_AGG(CommitID || ': ' || (NumberOfPass + NumberOfFail) || ': ' || COALESCE(ArtifactPath, ''), ', ') AS CommitCounts,
	STRING_AGG(CommitID || ': ' || TotalDuration || ': ' || COALESCE(ArtifactPath, ''), ', ') AS CommitDurations
	FROM lastn_env_data 
	GROUP BY StartOfDate
	ORDER BY StartOfDate DESC
	`
	var countsAndDurations []models.DBEnvDuration
	err = m.db.Select(&countsAndDurations, sqlQuer, envName, resolvedEnvGroup)
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
func (m *Postgres) GetOverview(dateRange int) (map[string]interface{}, error) {
	// dateRange is the number of days to use to look for "flaky-est" envs.
	start := time.Now()
	// Filters out old data and calculates the average number of failures and average duration per day per environment
	sqlQuery := `
	SELECT DATE_TRUNC('day', TestTime) AS StartOfDate, EnvName, EnvGroup, AVG(NumberOfFail) AS AvgFailedTests, AVG(TotalDuration) AS AvgDuration
	FROM db_environment_tests
	WHERE TestTime >= NOW() - INTERVAL '90 days'
	GROUP BY StartOfDate, EnvName, EnvGroup
	ORDER BY StartOfDate, EnvName, EnvGroup;
	`

	var summaryAvgFail []models.DBSummaryAvgFail
	err := m.db.Select(&summaryAvgFail, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query for summary chart: %v", err)
	}
	log.Printf("\nduration metric: took %f seconds to execute SQL query for summary duration and failure charts since start of handler", time.Since(start).Seconds())

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
	SELECT data.EnvName,
	data.EnvGroup,
	ROUND(COALESCE(AVG(CASE WHEN TestTime > (SELECT Date FROM recentCutoff) THEN NumberOfFail END), 0), 2) AS RecentNumberOfFail,
	ROUND(COALESCE(AVG(CASE WHEN TestTime <= (SELECT Date FROM recentCutoff) AND TestTime > (SELECT Date FROM prevCutoff) THEN NumberOfFail END), 0), 2) AS PrevNumberOfFail,
	COALESCE((SELECT TotalDuration FROM data As B WHERE B.EnvName=data.EnvName AND B.EnvGroup=data.EnvGroup ORDER BY TestTime DESC LIMIT 1),0)As TestDuration,
	COALESCE((SELECT TotalDuration FROM data As B WHERE B.EnvName=data.EnvName AND B.EnvGroup=data.EnvGroup ORDER BY TestTime DESC OFFSET 1 LIMIT 1),0)As PreviousTestDuration
	FROM data
	GROUP BY data.EnvName, data.EnvGroup
	ORDER BY RecentNumberOfFail DESC
	)
	SELECT EnvName, EnvGroup, RecentNumberOfFail, RecentNumberOfFail - PrevNumberOfFail AS Growth, TestDuration,PreviousTestDuration, TestDuration-PreviousTestDuration AS TestDurationGROWTH
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
