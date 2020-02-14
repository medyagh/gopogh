package report

import (
	"bytes"
	"encoding/json"
	"html/template"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/medyagh/gopogh/pkg/models"
)

// Version is gopogh version
const Version = "v0.1.0"

// Build includes commit sha date
var Build string

func mod(a, b int) int {
	return a % b
}

// DisplayContent represents the visible reporst to the end user
type DisplayContent struct {
	Results      map[string][]models.TestGroup
	TotalTests   int
	BuildVersion string
	CreatedOn    time.Time
	Report       models.Report
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
func Generate(report models.Report, groups []models.TestGroup) (DisplayContent, error) {
	var passedTests []models.TestGroup
	var failedTests []models.TestGroup
	var skippedTests []models.TestGroup
	for _, g := range groups {
		g.Duration = g.Events[len(g.Events)-1].Elapsed
		if !g.Hidden {
			if g.Status == "pass" {
				passedTests = append(passedTests, g)
			}
			if g.Status == "fail" {
				failedTests = append(failedTests, g)
			}
			if g.Status == "skip" {
				skippedTests = append(skippedTests, g)
			}

		}
	}

	testsNumber := len(passedTests) + len(failedTests) + len(skippedTests)
	rs := map[string][]models.TestGroup{}
	rs["pass"] = passedTests
	rs["fail"] = failedTests
	rs["skip"] = skippedTests
	return DisplayContent{Results: rs, TotalTests: testsNumber, BuildVersion: Version + "_" + Build, CreatedOn: time.Now(), Report: report}, nil
}
