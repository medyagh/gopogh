package report

import (
	"bytes"
	"html/template"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/medyagh/gopogh/pkg/models"
)

// Version is gopogh version
const Version = "v0.0.23"

// Build includes commit sha date
var Build string

func mod(a, b int) int {
	return a % b
}

// GenerateHTML geneates summerized html report
func GenerateHTML(report models.Report, groups []models.TestGroup) ([]byte, error) {
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

	type content struct {
		Results      map[string][]models.TestGroup
		TotalTests   int
		BuildVersion string
		CreatedOn    time.Time
		Report       models.Report
	}
	testsNumber := len(passedTests) + len(failedTests) + len(skippedTests)
	rs := map[string][]models.TestGroup{}
	rs["pass"] = passedTests
	rs["fail"] = failedTests
	rs["skip"] = skippedTests
	c := &content{Results: rs, TotalTests: testsNumber, BuildVersion: Version + "_" + Build, CreatedOn: time.Now(), Report: report}

	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}