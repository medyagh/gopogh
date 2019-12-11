package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	rice "github.com/GeertJohan/go.rice"
	"github.com/jstemmer/go-junit-report/parser"
)

type Results map[string][]*GoTestOutput

type GoTestOutput struct {
	PackageName string `json:"package_name"`
	TestName    string `json:"test_name"`
	Time        int    `json:"time"`
	Output      string `json:"output"`
}
type TestSummary struct {
	TotalTests int     `json:"total_tests"`
	Results    Results `json:"results"`
}

const (
	PASS = "pass"
	FAIL = "fail"
	SKIP = "skip"
	ALL  = "all"
)

var jsonTestKeys = map[parser.Result]string{
	parser.PASS: PASS,
	parser.FAIL: FAIL,
	parser.SKIP: SKIP,
}

func main() {
	out, err := os.Open("./testdata/minikube-logs.txt")
	if err != nil {
		log.Fatal(err)
	}
	xmlR, err := parser.Parse(out, "") // xml report
	if err != nil {
		log.Fatal(err)
	}

	totalTests := 0
	results := Results{
		ALL:  []*GoTestOutput{},
		PASS: []*GoTestOutput{},
		FAIL: []*GoTestOutput{},
		SKIP: []*GoTestOutput{},
	}
	for _, pkg := range xmlR.Packages {
		fmt.Println("------------------")
		fmt.Println(pkg.Name)
		fmt.Println(pkg.Duration)
		for _, t := range pkg.Tests {
			key := jsonTestKeys[t.Result]
			fmt.Printf("\nt.Name=%s , t.Result=%s, t.Time=%d\n", t.Name, key, t.Time)
			jsonTest := &GoTestOutput{
				PackageName: pkg.Name,
				TestName:    t.Name,
				Time:        t.Time,
				Output:      strings.Join(t.Output, "\n"),
			}
			results[key] = append(results[key], jsonTest)
			results[ALL] = append(results[ALL], jsonTest)
			totalTests += 1
		}
		fmt.Println("------------------")
	}

	summary := &TestSummary{
		TotalTests: totalTests,
		Results:    results,
	}

	html, err := generateHTML(summary)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile("output.html", []byte(html), 0644)
	if err != nil {
		log.Fatal(err)
	}

}

func generateHTML(summary *TestSummary) (string, error) {
	templateBox := rice.MustFindBox("template")
	t, err := template.New("report").Parse(templateBox.MustString("report.html"))
	if err != nil {
		return "", err
	}

	type templateData struct {
		Summary *TestSummary
	}
	buf := bytes.Buffer{}
	err = t.Execute(&buf, &templateData{Summary: summary})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
