package out

import (
	"bytes"
	"html/template"

	rice "github.com/GeertJohan/go.rice"
	"github.com/medyagh/goprettyorgohome/models"
)

// GenerateHTML geneates summerized html report
func GenerateHTML(groups []models.TestGroup) ([]byte, error) {
	t, err := template.New("out").Parse(rice.MustFindBox("../template").MustString("report3.html"))
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
		Pass        []models.TestGroup
		Fail        []models.TestGroup
		Skip        []models.TestGroup
		ResultTypes []template.JS
		TotalTests  int
	}
	testsNumber := len(passedTests) + len(failedTests) + len(skippedTests)
	c := &content{Pass: passedTests, Fail: failedTests, Skip: skippedTests, ResultTypes: []template.JS{"fail", "pass", "skip"}, TotalTests: testsNumber}
	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
