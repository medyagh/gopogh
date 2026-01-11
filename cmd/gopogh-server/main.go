// Package main provides the gopogh-server command
package main

import (
	_ "embed"
	"flag"
	"log"
	"net/http"

	"github.com/medyagh/gopogh/pkg/db"
	"github.com/medyagh/gopogh/pkg/handler"
)

var dbPath = flag.String("db_path", "", "path to postgres db in the form of 'user=DB_USER dbname=DB_NAME password=DB_PASS'")
var dbHost = flag.String("db_host", "", "host of the db")
var useCloudSQL = flag.Bool("use_cloudsql", false, "whether the database is a cloudsql db")
var useIAMAuth = flag.Bool("use_iam_auth", false, "whether to use IAM to authenticate with the cloudsql db")

func main() {
	flag.Parse()
	flagValues := db.FlagValues{
		Backend:     "postgres",
		Host:        *dbHost,
		Path:        *dbPath,
		UseCloudSQL: *useCloudSQL,
		UseIAMAuth:  *useIAMAuth,
	}
	datab, err := db.FromEnv(flagValues)
	if err != nil {
		log.Fatal(err)
	}
	db := handler.DB{
		Database: datab,
	}
	// Create an HTTP server and register the handlers

	http.HandleFunc("/db", db.ServeEnvironmentTestsAndTestCases)

	http.HandleFunc("/env", db.ServeEnvCharts)

	http.HandleFunc("/test", db.ServeTestCharts)

	http.HandleFunc("/summary", db.ServeOverview)

	http.HandleFunc("/version", handler.ServeGopoghVersion)

	http.HandleFunc("/load-testgrid", db.LoadTestGrid)

	http.HandleFunc("/", handler.ServeHTML)

	// Start the HTTP server
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}
