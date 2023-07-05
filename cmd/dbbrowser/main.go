package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/medyagh/gopogh/pkg/db"
)

var dbPath = flag.String("db_path", "", "path to cloudsql db in the form of 'host=HOST_NAME user=DB_USER dbname=DB_NAME password=DB_PASS'")

func main() {
	flag.Parse()
	if *dbPath == "" {
		log.Fatal(fmt.Errorf("db_path not specified"))
	}
	cfg := db.Config{
		Type: "cloudsql",
		Path: *dbPath,
	}
	pg, err := db.NewCloudPostgres(cfg)
	if err != nil {
		log.Fatal(err)
	}
	// Create an HTTP server and register the handlers
	http.HandleFunc("/db", pg.PrintEnvironmentTestsAndTestCases)

	// Start the HTTP server
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}
