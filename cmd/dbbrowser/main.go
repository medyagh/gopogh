package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/medyagh/gopogh/pkg/db"
)

var dbPath = flag.String("db_path", "", "path to postgres db in the form of 'host=HOST_NAME user=DB_USER dbname=DB_NAME password=DB_PASS'")
var useCloudSQL = flag.Bool("use_cloudsql", false, "whether the database is a cloudsql db")

func main() {
	flag.Parse()
	if *dbPath == "" {
		log.Fatalf("db_path not specified")
	}
	db, err := db.FromEnv(*dbPath, "postgres", *useCloudSQL)
	if err != nil {
		log.Fatal(err)
	}
	// Create an HTTP server and register the handlers
	http.HandleFunc("/db", db.PrintEnvironmentTestsAndTestCases)

	// Start the HTTP server
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}
