package main

import (
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
	if *dbPath == "" {
		log.Fatalf("The db_path flag is required")
	}
	if *dbHost == "" {
		log.Fatalf("The db_host flag is required")
	}
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
	db := handler.HandlerDB{
		Database: datab,
	}
	// Create an HTTP server and register the handlers
	http.HandleFunc("/db", db.ServeEnvironmentTestsAndTestCases)

	http.HandleFunc("/env", db.ServeEnvCharts)

	http.HandleFunc("/test", db.ServeTestCharts)

	http.HandleFunc("/summary", db.ServeOverview)

	// Start the HTTP server
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}
