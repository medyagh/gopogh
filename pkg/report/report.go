package report

import (
	"bytes"
	"encoding/json"
	"html/template"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/medyagh/gopogh/pkg/models"
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
		NumberOfFail  int
		NumberOfPass  int
		NumberOfSkip  int
		FailedTests   []string
		GopoghVersion string
		GopoghBuild   string
		Detail        models.ReportDetail
	}
	ss := shortSummary{}
	for _, t := range resultTypes {
		if t == pass {
			ss.NumberOfPass = len(c.Results[t])
		}
		if t == fail {
			ss.NumberOfFail = len(c.Results[t])
			for _, t := range c.Results[t] {
				ss.FailedTests = append(ss.FailedTests, t.TestName)
			}
		}
		if t == skip {
			ss.NumberOfSkip = len(c.Results[t])
		}

	}
	ss.Detail = c.Detail
	ss.GopoghVersion = Version
	ss.GopoghBuild = Build
	return json.MarshalIndent(ss, "", "    ")
}

// JSON return the report in json
func (c DisplayContent) JSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "    ")
}

// HTML returns html format
func (c DisplayContent) HTML() ([]byte, error) {

	fmap := template.FuncMap{
		"mod": mod,
	}
	t, err := template.New("out").Parse(rice.MustFindBox("../template").MustString("report3.css"))
	if err != nil {
		return nil, err
	}

	t, err = t.Funcs(fmap).Parse(rice.MustFindBox("../template").MustString("report3.html"))
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Generate geneates a report
func Generate(report models.ReportDetail, groups []models.TestGroup) (DisplayContent, error) {
	var passedTests []models.TestGroup
	var failedTests []models.TestGroup
	var skippedTests []models.TestGroup
	for _, g := range groups {
		g.Duration = g.Events[len(g.Events)-1].Elapsed
		if !g.Hidden {
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
