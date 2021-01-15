package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/medyagh/gopogh/pkg/models"
	"github.com/medyagh/gopogh/pkg/parser"
	"github.com/medyagh/gopogh/pkg/report"
)

// Build includes commit sha date
var Build string

var (
	reportName     = flag.String("name", "", "report name ")
	reportPR       = flag.String("pr", "", "Pull request number")
	reportDetails  = flag.String("details", "", "report details (for example test args...)")
	reportRepo     = flag.String("repo", "", "source repo")
	inPath         = flag.String("in", "", "path to JSON file produced by go tool test2json")
	outPath        = flag.String("out", "", "(depricated use  -out_html instead) path to HTML output file ")
	outHTMLPath    = flag.String("out_html", "", "path to HTML output file")
	outSummaryPath = flag.String("out_summary", "", "path to json summary output file")
	version        = flag.Bool("version", false, "shows version")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Printf("Version %s Build %s", report.Version, report.Build)
	}

	if *inPath == "" {
		panic("must provide path to JSON input file")
	}

	if *outPath != "" {
		*outHTMLPath = *outPath
	}

	if *outHTMLPath == "" {
		panic("must provide path to HTML output file")
	}

	events, err := parser.ParseJSON(*inPath)
	if err != nil {
		panic(fmt.Sprintf("json: %v", err))
	}
	groups := parser.ProcessEvents(events)
	r := models.ReportDetail{Name: *reportName, Details: *reportDetails, PR: *reportPR, RepoName: *reportRepo}
	c, err := report.Generate(r, groups)
	if err != nil {
		panic(fmt.Sprintf("failed to generate report: %v", err))
	}

	html, err := c.HTML()
	if err != nil {
		fmt.Printf("failed to convert report to html: %v", err)
	} else {
		if err := ioutil.WriteFile(*outHTMLPath, html, 0644); err != nil {
			panic(fmt.Sprintf("failed to write the html output %s: %v", *outHTMLPath, err))
		}
	}
	j, err := c.ShortSummary()
	if err != nil {
		fmt.Printf("failed to convert report to json: %v", err)
	} else {
		if *outSummaryPath != "" {
			if err := ioutil.WriteFile(*outSummaryPath, j, 0644); err != nil {
				panic(fmt.Sprintf("failed to write the html output %s: %v", *outSummaryPath, err))
			}
		}
		fmt.Println(string(j))
	}
}
