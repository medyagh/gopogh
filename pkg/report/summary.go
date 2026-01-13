package report

import (
	"fmt"
	"time"

	"github.com/medyagh/gopogh/pkg/models"
)

// Summary mirrors the JSON output from DisplayContent.ShortSummary.
type Summary struct {
	NumberOfTests int                 `json:"NumberOfTests"`
	NumberOfFail  int                 `json:"NumberOfFail"`
	NumberOfPass  int                 `json:"NumberOfPass"`
	NumberOfSkip  int                 `json:"NumberOfSkip"`
	FailedTests   []string            `json:"FailedTests"`
	PassedTests   []string            `json:"PassedTests"`
	SkippedTests  []string            `json:"SkippedTests"`
	Durations     map[string]float64  `json:"Durations"`
	TotalDuration float64             `json:"TotalDuration"`
	GopoghVersion string              `json:"GopoghVersion"`
	GopoghBuild   string              `json:"GopoghBuild"`
	Detail        models.ReportDetail `json:"Detail"`
}

// Validate ensures the summary looks like a gopogh ShortSummary payload.
func (s Summary) Validate() error {
	if s.Detail.Name == "" {
		return fmt.Errorf("missing detail name")
	}
	if s.Detail.Details == "" {
		return fmt.Errorf("missing detail id")
	}
	totalFromSlices := len(s.FailedTests) + len(s.PassedTests) + len(s.SkippedTests)
	if totalFromSlices == 0 {
		return fmt.Errorf("no tests listed")
	}
	if s.NumberOfFail != len(s.FailedTests) {
		return fmt.Errorf("failed count mismatch: expected %d, got %d", len(s.FailedTests), s.NumberOfFail)
	}
	if s.NumberOfPass != len(s.PassedTests) {
		return fmt.Errorf("pass count mismatch: expected %d, got %d", len(s.PassedTests), s.NumberOfPass)
	}
	if s.NumberOfSkip != len(s.SkippedTests) {
		return fmt.Errorf("skip count mismatch: expected %d, got %d", len(s.SkippedTests), s.NumberOfSkip)
	}
	if s.NumberOfTests != s.NumberOfFail+s.NumberOfPass+s.NumberOfSkip {
		return fmt.Errorf("total test count mismatch: expected %d, got %d", s.NumberOfFail+s.NumberOfPass+s.NumberOfSkip, s.NumberOfTests)
	}
	return nil
}

// ToDBRows converts a valid summary into database rows.
func (s Summary) ToDBRows(testTime time.Time) (models.DBEnvironmentTest, []models.DBTestCase, error) {
	if err := s.Validate(); err != nil {
		return models.DBEnvironmentTest{}, nil, err
	}
	dbRows := make([]models.DBTestCase, 0, s.NumberOfTests)
	addRows := func(result string, tests []string) {
		for _, testName := range tests {
			duration := 0.0
			if s.Durations != nil {
				if d, ok := s.Durations[testName]; ok {
					duration = d
				}
			}
			dbRows = append(dbRows, models.DBTestCase{
				PR:       s.Detail.PR,
				CommitID: s.Detail.Details,
				TestName: testName,
				Result:   result,
				Duration: duration,
				EnvName:  s.Detail.Name,
				TestTime: testTime,
			})
		}
	}
	addRows(pass, s.PassedTests)
	addRows(fail, s.FailedTests)
	addRows(skip, s.SkippedTests)

	dbEnvironmentRow := models.DBEnvironmentTest{
		CommitID:      s.Detail.Details,
		EnvName:       s.Detail.Name,
		GopoghTime:    time.Now(),
		TestTime:      testTime,
		NumberOfFail:  s.NumberOfFail,
		NumberOfPass:  s.NumberOfPass,
		NumberOfSkip:  s.NumberOfSkip,
		TotalDuration: s.TotalDuration,
		GopoghVersion: s.GopoghVersion,
	}
	return dbEnvironmentRow, dbRows, nil
}
