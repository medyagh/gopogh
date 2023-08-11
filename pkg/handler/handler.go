package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/medyagh/gopogh/pkg/db"
	"github.com/medyagh/gopogh/pkg/report"
)

type DB struct {
	Database db.Datab
}

func (m *DB) ServeEnvironmentTestsAndTestCases(w http.ResponseWriter, _ *http.Request) {
	data, err := m.Database.GetEnvironmentTestsAndTestCases()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, "Failed to write JSON data", http.StatusInternalServerError)
		return
	}
}

// ServeTestCharts writes the individual test charts to a JSON HTTP response
func (m *DB) ServeTestCharts(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	env := queryValues.Get("env")
	if env == "" {
		http.Error(w, "missing environment name", http.StatusUnprocessableEntity)
		return
	}
	test := queryValues.Get("test")
	if test == "" {
		http.Error(w, "missing test name", http.StatusUnprocessableEntity)
		return
	}

	data, err := m.Database.GetTestCharts(env, test)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, "Failed to write JSON data", http.StatusInternalServerError)
		return
	}
}

// ServeEnvCharts writes the overall environment charts to a JSON HTTP response
func (m *DB) ServeEnvCharts(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	env := queryValues.Get("env")
	if env == "" {
		http.Error(w, "missing environment name", http.StatusUnprocessableEntity)
		return
	}
	testsInTopStr := queryValues.Get("tests_in_top")
	if testsInTopStr == "" {
		testsInTopStr = "10"
	}
	testsInTop, err := strconv.Atoi(testsInTopStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid number of top tests to use: %v", err), http.StatusUnprocessableEntity)
		return
	}
	data, err := m.Database.GetEnvCharts(env, testsInTop)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, "Failed to write JSON data", http.StatusInternalServerError)
		return
	}
}

// ServeOverview writes the overview chart for all of the environments to a JSON HTTP response
func (m *DB) ServeOverview(w http.ResponseWriter, _ *http.Request) {
	data, err := m.Database.GetOverview()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, "Failed to write JSON data", http.StatusInternalServerError)
		return
	}
}

// ServeGopoghVersion writes the gopogh version to a json response
func ServeGopoghVersion(w http.ResponseWriter, _ *http.Request) {
	data := map[string]interface{}{
		"version": report.Version,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, "Failed to write JSON data", http.StatusInternalServerError)
		return
	}
}
