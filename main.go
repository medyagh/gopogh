package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/medyagh/gopogh/out"
	"github.com/medyagh/gopogh/parser"
)

var (
	reportName     = flag.String("name", "", "report name ")
	reportPR       = flag.String("pr", "", "Pull request number")
	reportDetails  = flag.String("details", "", "report details (for example test args...)")
	reportRepo     = flag.String("repo", "", "source repo")
	inPath         = flag.String("in", "", "path to JSON input file")
	outPath        = flag.String("out", "", "path to HTML output file")
	summaryOutPath = flag.String("summary", "", "path to summary output file")
	statusOutPath  = flag.String("status", "", "path to status output file")
	version        = flag.Bool("version", false, "shows version")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Printf("Version %s Build %s", out.Version, out.Build)
		return
	}

	if *inPath == "" {
		panic("must provide path to JSON input file")
	}
	if *outPath == "" {
		panic("must provide path to HTML output file")
	}

	rCfg := out.ReportConfig{Name: *reportName, Details: *reportDetails, PR: *reportPR, RepoName: *reportRepo}

	events, err := parser.LoadGoJSON(*inPath)
	if err != nil {
		panic(fmt.Sprintf("json: %v", err))
	}

	report := parser.GenerateDetailedReport(events)

	html, err := out.GenerateHTML(rCfg, report)
	if err != nil {
		panic(fmt.Sprintf("html: %v", err))
	}
	if err := ioutil.WriteFile(*outPath, html, 0644); err != nil {
		panic(fmt.Sprintf("write: %v", err))
	}
	status := report.PessemisticStatus()
	if *statusOutPath != "" {
		if err := ioutil.WriteFile(*summaryOutPath, []byte(status), 0644); err != nil {
			fmt.Printf("error writing status %v", err)
		}
	}

	fmt.Println(status)

	if *summaryOutPath != "" {
		s, err := json.Marshal(report.Summary())
		if err != nil {
			fmt.Printf("error marshalizing summary: %v", err)
		}
		if err := ioutil.WriteFile(*summaryOutPath, s, 0644); err != nil {
			panic(fmt.Sprintf("write: %v", err))
		}
	}

}
