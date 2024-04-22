package models

import "time"

// ReportDetail holds the report details such as test name, PR number...
type ReportDetail struct {
	Name     string
	Details  string
	PR       string // pull request number
	RepoName string // for example github repo
}
type TestEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string

	EmbeddedLog []string
}

type TestGroup struct {
	TestName  string
	TestOrder int
	Hidden    bool
	Status    string
	Start     time.Time
	End       time.Time
	Duration  float64
	Events    []TestEvent
}

// DBTestCase represents a row in db table that holds each individual subtest
type DBTestCase struct {
	PR        string
	CommitID  string
	TestName  string
	TestTime  time.Time
	Result    string
	Duration  float64
	EnvName   string
	TestOrder int
}

// DBEnvironmentTest represents a row in db table that has finished tests in each environment
type DBEnvironmentTest struct {
	CommitID      string
	EnvName       string
	GopoghTime    time.Time
	TestTime      time.Time
	NumberOfFail  int
	NumberOfPass  int
	NumberOfSkip  int
	TotalDuration float64
	GopoghVersion string
}

// DBFlakeRow represents a row in the basic flake rate table
type DBFlakeRow struct {
	TestName              string  `json:"testName"`
	RecentFlakePercentage float32 `json:"recentFlakePercentage"`
	GrowthRate            float32 `json:"growthRate"`
}

// DBFlakeBy represents a "row" in the flake rate by _ of top 10 of recent test flakiness charts
type DBFlakeBy struct {
	TestName        string    `json:"testName"`
	StartOfDate     time.Time `json:"startOfDate"`
	FlakePercentage float32   `json:"flakePercentage"`
	CommitResults   string    `json:"commitResults"`
}

// DBEnvDuration represents a "row" in the test count and total duration by day chart
type DBEnvDuration struct {
	StartOfDate     time.Time `json:"startOfDate"`
	TestCount       float32   `json:"testCount"`
	Duration        float32   `json:"duration"`
	CommitCounts    string    `json:"commitCounts"`
	CommitDurations string    `json:"commitDurations"`
}

// DBTestRateAndDuration represents a "row" in the flake rate and duration chart for a given test
type DBTestRateAndDuration struct {
	StartOfDate               time.Time `json:"startOfDate"`
	AvgDuration               float32   `json:"avgDuration"`
	FlakePercentage           float32   `json:"flakePercentage"`
	CommitResultsAndDurations string    `json:"commitResultsAndDurations"`
}

// DBSummaryAvgFail represents a "row" in most flakey environments summary chart
type DBSummaryAvgFail struct {
	StartOfDate    time.Time `json:"startOfDate"`
	EnvName        string    `json:"envName"`
	AvgFailedTests float32   `json:"avgFailedTests"`
	AvgDuration    float32   `json:"avgDuration"`
}

// DBSummaryTable represents a row in the summary number of fail table
type DBSummaryTable struct {
	EnvName              string  `json:"envName"`
	RecentNumberOfFail   float32 `json:"recentNumberOfFail"`
	Growth               float32 `json:"growth"`
	TestDuration         float32 `json:"testDuration"`
	PreviousTestDuration float32 `json:"previousTestDuration"`
	TestDurationGrowth   float32 `json:"testDurationGrowth"`
}
