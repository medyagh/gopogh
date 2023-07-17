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
	TestTime  string
	Result    string
	Duration  float64
	EnvName   string
	TestOrder int
}

// DBEnvironmentTest represents a row in db table that has finished tests in each environment
type DBEnvironmentTest struct {
	CommitID      string
	EnvName       string
	GopoghTime    string
	TestTime      string
	NumberOfFail  int
	NumberOfPass  int
	NumberOfSkip  int
	TotalDuration float64
	GopoghVersion string
}
