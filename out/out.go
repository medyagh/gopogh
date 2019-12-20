package out

import (
	"bytes"
	"html/template"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/medyagh/gopogh/parser"
)

const Version = "v0.0.15" // Version is gopogh version

var (
	Build string //  commitsha injected during build
)

func mod(a, b int) int {
	return a % b
}

// ReportConfig holds the report details such as test name, PR number...
type ReportConfig struct {
	Name     string
	Details  string
	PR       string // pull request number
	RepoName string // for example github repo
}

// GenerateHTML returns HTML bytes out of a report
func GenerateHTML(cfg ReportConfig, r parser.DetailedReport) ([]byte, error) {
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

	type content struct {
		Results      map[string][]parser.TestGroup
		TotalTests   int
		BuildVersion string
		CreatedOn    time.Time
		Report       ReportConfig
	}
	testsNumber := r.TotalTests
	rs := map[string][]parser.TestGroup{}
	rs["pass"] = r.PassedTests
	rs["fail"] = r.FailedTests
	rs["skip"] = r.SkippedTests
	c := &content{Results: rs, TotalTests: testsNumber, BuildVersion: Version + "_" + Build, CreatedOn: time.Now(), Report: cfg}

	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
