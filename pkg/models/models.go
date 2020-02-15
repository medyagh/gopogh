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
	TestName string
	Hidden   bool
	Status   string
	Start    time.Time
	End      time.Time
	Duration float64
	Events   []TestEvent
}
