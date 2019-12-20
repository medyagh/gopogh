package parser

import (
	"time"
)

// GenerateDetailedReport categorizes and calculates durations
func GenerateDetailedReport(events []TestEvent) DetailedReport {
	groups := processEvents(events)
	var p []TestGroup
	var f []TestGroup
	var s []TestGroup
	var all = 0
	for _, g := range groups {
		g.Duration = g.Events[len(g.Events)-1].Elapsed
		if !g.Hidden {
			all = all + 1
			if g.Status == "pass" {
				p = append(p, g)
			}
			if g.Status == "fail" {
				f = append(f, g)
			}
			if g.Status == "skip" {
				s = append(s, g)
			}

		}
	}
	return DetailedReport{PassedTests: p, FailedTests: f, SkippedTests: s, TotalTests: all}
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

type TestSummary struct {
	TestName string
	Duration float64
}

type DetailedReport struct {
	TotalTests   int
	PassedTests  []TestGroup
	FailedTests  []TestGroup
	SkippedTests []TestGroup
}

type SummaryReport struct {
	TotalTests   int
	PassedTests  []TestSummary
	FailedTests  []TestSummary
	SkippedTests []TestSummary
}

func (d DetailedReport) Summary() SummaryReport {
	sr := SummaryReport{}
	for _, p := range d.PassedTests {
		sr.PassedTests = append(sr.PassedTests, TestSummary{TestName: p.TestName, Duration: p.Duration})
	}
	for _, p := range d.FailedTests {
		sr.FailedTests = append(sr.FailedTests, TestSummary{TestName: p.TestName, Duration: p.Duration})
	}
	for _, p := range d.SkippedTests {
		sr.SkippedTests = append(sr.SkippedTests, TestSummary{TestName: p.TestName, Duration: p.Duration})
	}
	return sr
}
