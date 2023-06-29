package main

import (
	"log"
	"net/http"
	"os"

	"github.com/medyagh/gopogh/pkg/db"
)

func main() {
	path := "host=k8s-minikube:us-west1:flake-rate user=postgres dbname=flakedbdev password="
	path = path + os.Getenv("DB_PASS")
	cfg := db.Config{
		Type: "cloudsql",
		Path: path,
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
