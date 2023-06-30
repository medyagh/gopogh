package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/medyagh/gopogh/pkg/db"
)

var dbPath = flag.String("db_path", "", "path to cloudsql db in the form of 'host=HOST_NAME user=DB_USER dbname=DB_NAME password=DB_PASS'")

func main() {
	flag.Parse()
	if *dbPath == "" {
		fmt.Println("db_path not specified. defaulting to minikube")
		mkPath := "host=k8s-minikube:us-west1:flake-rate user=postgres dbname=flakedbdev password="
		mkPath += os.Getenv("DB_PASS")
		*dbPath = mkPath
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
