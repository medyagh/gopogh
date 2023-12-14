package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/medyagh/gopogh/pkg/db"
	"github.com/medyagh/gopogh/pkg/models"
	"github.com/medyagh/gopogh/pkg/parser"
	"github.com/medyagh/gopogh/pkg/report"
)

// Build includes commit sha date
var Build string

var (
	dbBackend      = flag.String("db_backend", "", "sql database driver. 'sqlite' for file output")
	dbHost         = flag.String("db_host", "", "host of the db")
	dbPath         = flag.String("db_path", "", "path to sql database/database file. if using postgres in the form of 'user=DB_USER dbname=DB_NAME password=DB_PASS'")
	useCloudSQL    = flag.Bool("use_cloudsql", false, "whether the database is a cloudsql db")
	useIAMAuth     = flag.Bool("use_iam_auth", false, "whether to use IAM to authenticate with the cloudsql db")
	reportName     = flag.String("name", "", "report name")
	reportPR       = flag.String("pr", "", "Pull request number")
	reportDetails  = flag.String("details", "", "report details (for example test args...)")
	reportRepo     = flag.String("repo", "", "source repo")
	inPath         = flag.String("in", "", "path to JSON file produced by go tool test2json")
	outPath        = flag.String("out", "", "(deprecated use  -out_html instead) path to HTML output file")
	outHTMLPath    = flag.String("out_html", "", "path to HTML output file")
	outSummaryPath = flag.String("out_summary", "", "path to json summary output file")
	version        = flag.Bool("version", false, "shows version")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Printf("Version %s Build %s", report.Version(), report.Build)
	}

	if *inPath == "" {
		fmt.Println("Please provide path to JSON input file using -in")
		os.Exit(1)
	}

	if *outPath != "" {
		*outHTMLPath = *outPath
	}

	if *outHTMLPath == "" {
		fmt.Println("Please provide path to HTML output file using -out_html")
		os.Exit(1)
	}

	events, err := parser.ParseJSON(*inPath)
	if err != nil {
		fmt.Printf("json: %v", err)
		os.Exit(1)
	}
	groups := parser.ProcessEvents(events)
	r := models.ReportDetail{Name: *reportName, Details: *reportDetails, PR: *reportPR, RepoName: *reportRepo}
	c, err := report.Generate(r, groups)
	if err != nil {
		fmt.Printf("failed to generate report: %v", err)
		os.Exit(1)
	}

	if dbVarProvided(*dbPath, *dbBackend, *dbHost) {
		flagValues := db.FlagValues{
			Backend:     *dbBackend,
			Host:        *dbHost,
			Path:        *dbPath,
			UseCloudSQL: *useCloudSQL,
			UseIAMAuth:  *useIAMAuth,
		}
		if err := c.SQL(flagValues); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	html, err := c.HTML()
	if err != nil {
		fmt.Printf("failed to convert report to html: %v", err)
	} else {
		if err := os.MkdirAll(filepath.Dir(*outHTMLPath), 0755); err != nil {
			fmt.Printf("failed to create directory: %v", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*outHTMLPath, html, 0644); err != nil {
			fmt.Printf("failed to write the html output %s: %v", *outHTMLPath, err)
			os.Exit(1)
		}
	}
	j, err := c.ShortSummary()
	if err != nil {
		fmt.Printf("failed to convert report to json: %v", err)
	} else {
		if *outSummaryPath != "" {
			if err := os.MkdirAll(filepath.Dir(*outSummaryPath), 0755); err != nil {
				fmt.Printf("failed to create directory: %v", err)
				os.Exit(1)
			}
			if err := os.WriteFile(*outSummaryPath, j, 0644); err != nil {
				fmt.Printf("failed to write the html output %s: %v", *outSummaryPath, err)
				os.Exit(1)
			}
		}
		fmt.Println(string(j))
	}
}

// dbVarProvided checks whether any of the database flags/environment variables are set
func dbVarProvided(dbPath, dbBackend, dbHost string) bool {
	values := []string{
		dbBackend,
		dbHost,
		dbPath,
		os.Getenv("DB_BACKEND"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PATH"),
	}
	for _, v := range values {
		if v != "" {
			return true
		}
	}
	return false
}
