package report

import (
	"bytes"
	"encoding/json"
	"html/template"
	"time"

	"github.com/medyagh/gopogh/pkg/models"
	"github.com/medyagh/gopogh/pkg/templates"
)

// DisplayContent represents the visible reporst to the end user
type DisplayContent struct {
	Results      map[string][]models.TestGroup
	TotalTests   int
	BuildVersion string
	CreatedOn    time.Time
	Detail       models.ReportDetail
}

// ShortSummary returns only test names without logs
func (c DisplayContent) ShortSummary() ([]byte, error) {
	type shortSummary struct {
		NumberOfTests int
		NumberOfFail  int
		NumberOfPass  int
		NumberOfSkip  int
		FailedTests   []string
		PassedTests   []string
		SkippedTests  []string
		Durations     map[string]float64
		GopoghVersion string
		GopoghBuild   string
		Detail        models.ReportDetail
	}
	ss := shortSummary{}
	ss.Durations = make(map[string]float64)
	for _, t := range resultTypes {
		if t == pass {
			ss.NumberOfPass = len(c.Results[t])
			for _, ti := range c.Results[t] {
				ss.PassedTests = append(ss.PassedTests, ti.TestName)
				ss.Durations[ti.TestName] = ti.Duration
			}
		}
		if t == fail {
			ss.NumberOfFail = len(c.Results[t])
			for _, ti := range c.Results[t] {
				ss.FailedTests = append(ss.FailedTests, ti.TestName)
				ss.Durations[ti.TestName] = ti.Duration
			}
		}
		if t == skip {
			ss.NumberOfSkip = len(c.Results[t])
			for _, ti := range c.Results[t] {
				ss.SkippedTests = append(ss.SkippedTests, ti.TestName)
				// not adding to the skip test durations to avoid confusion or bad data, since they will be 0seconds most-likely
				// but if I change my mind we need to uncomment this line
				// ss.Durations[ti.TestName] = ti.Duration
			}
		}

	}
	ss.NumberOfTests = ss.NumberOfFail + ss.NumberOfPass + ss.NumberOfSkip
	ss.Detail = c.Detail
	ss.GopoghVersion = Version
	ss.GopoghBuild = Build
	return json.MarshalIndent(ss, "", "    ")
}

// HTML returns html format
func (c DisplayContent) HTML() ([]byte, error) {

	fmap := template.FuncMap{
		"mod": mod,
	}
	t, err := template.New("out").Parse(templates.ReportCSS)
	if err != nil {
		return nil, err
	}

	t, err = t.Funcs(fmap).Parse(templates.ReportHTML)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Generate generates a report
func Generate(report models.ReportDetail, groups []models.TestGroup) (DisplayContent, error) {
	var passedTests []models.TestGroup
	var failedTests []models.TestGroup
	var skippedTests []models.TestGroup
	order := 0
	for _, g := range groups {
		order = order + 1
		g.Duration = g.Events[len(g.Events)-1].Elapsed
		if !g.Hidden {
			g.TestOrder = order
			if g.Status == pass {
				passedTests = append(passedTests, g)
			}
			if g.Status == fail {
				failedTests = append(failedTests, g)
			}
			if g.Status == skip {
				skippedTests = append(skippedTests, g)
			}

		}
	}

	testsNumber := len(passedTests) + len(failedTests) + len(skippedTests)
	rs := map[string][]models.TestGroup{}
	rs[pass] = passedTests
	rs[fail] = failedTests
	rs[skip] = skippedTests
	return DisplayContent{Results: rs, TotalTests: testsNumber, BuildVersion: Version + "_" + Build, CreatedOn: time.Now(), Detail: report}, nil
}

func mod(a, b int) int {
	return a % b
}
