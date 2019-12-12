package models

import "time"

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
	Test     string
	Hidden   bool
	Status   string
	Start    time.Time
	End      time.Time
	Duration float64
	Events   []TestEvent
}
