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

	type content struct {
		TestName string
		Duration string
		Result   string
		Events   string
	}

	var contents []content
	for _, g := range groups {
		d := fmt.Sprintf("%f", g.Events[len(g.Events)-1].Elapsed)
		logs := ""
		for _, l := range g.Events {
			logs = logs + "\n" + l.Output
		}
		if !g.Hidden {
			contents = append(contents, content{TestName: g.Test, Duration: d, Result: g.Status, Events: logs})
		}
	}

	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", contents); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
