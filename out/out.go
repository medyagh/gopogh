package out

import (
	"bytes"
	"html/template"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/medyagh/gopogh/models"
)

const Version = "v0.0.15" // Version is gopogh version

var (
	Build string //  commitsha injected during build
)

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
