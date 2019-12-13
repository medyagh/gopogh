package out

import (
	"bytes"
	"fmt"
	"html/template"

	rice "github.com/GeertJohan/go.rice"
	"github.com/medyagh/goprettyorgohome/models"
)

// GenerateHTML geneates summerized html report
func GenerateHTML(groups []models.TestGroup) ([]byte, error) {
	for _, g := range groups {
		g.Duration = g.Events[len(g.Events)-1].Elapsed
	}
	t, err := template.New("out").Parse(rice.MustFindBox("../template").MustString("report.html"))
	if err != nil {
		return nil, err
	}

	type testRow struct {
		TestName string
		Duration string
		Result   string
		Events   string
	}

	var passedTests []testRow
	var failedTests []testRow
	var skippedTests []testRow
	for _, g := range groups {
		d := fmt.Sprintf("%f", g.Events[len(g.Events)-1].Elapsed)
		logs := ""
		for _, l := range g.Events {
			logs = logs + "\n" + l.Output
		}
		if !g.Hidden {
			if g.Status == "pass" {
				passedTests = append(passedTests, testRow{TestName: g.Test, Duration: d, Result: g.Status, Events: logs})
			}
			if g.Status == "fail" {
				failedTests = append(failedTests, testRow{TestName: g.Test, Duration: d, Result: g.Status, Events: logs})
			}
			if g.Status == "skip" {
				skippedTests = append(skippedTests, testRow{TestName: g.Test, Duration: d, Result: g.Status, Events: logs})
			}

		}
	}

	type content struct {
		Pass        []testRow
		Fail        []testRow
		Skip        []testRow
		ResultTypes []string
	}

	c := &content{Pass: passedTests, Fail: failedTests, Skip: skippedTests, ResultTypes: []string{"pass", "fail", "skip"}}
	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
