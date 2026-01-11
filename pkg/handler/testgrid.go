package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/medyagh/gopogh/pkg/report"
	"github.com/medyagh/testgrid-crawler/pkg/crawler"
)

const (
	testgridDashboard    = "minikube-periodics#ci-minikube-integration"
	testgridJobName      = "ci-minikube-integration"
	defaultMaxPages      = 20
	defaultConcurrency   = 6
	maxErrorSampleCount  = 10
	summaryFetchTimeout  = 20 * time.Second
)

var errSummaryNotFound = errors.New("summary not found")

const testgridLoaderHTML = `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Load TestGrid</title>
    <style>
      body { font-family: Arial, sans-serif; margin: 2rem; }
      button { padding: 0.5rem 1rem; }
      #status { margin-left: 1rem; }
    </style>
  </head>
  <body>
    <h2>Load TestGrid Data</h2>
    <p>Dashboard: minikube-periodics#ci-minikube-integration</p>
    <button id="loadBtn">Load TestGrid</button>
    <span id="status"></span>
    <p><a href="/">Back to charts</a></p>
    <script>
      const button = document.getElementById("loadBtn");
      const status = document.getElementById("status");
      button.addEventListener("click", async () => {
        button.disabled = true;
        status.textContent = "Loading...";
        try {
          const response = await fetch("/load-testgrid", { method: "POST" });
          if (!response.ok) {
            throw new Error("Server returned " + response.status);
          }
          const data = await response.json();
          const errorSample = data.errorSamples && data.errorSamples.length ? " Sample error: " + data.errorSamples[0] : "";
          status.textContent = "Loaded " + data.inserted + " jobs. Missing: " + data.missingSummary +
            ", Invalid: " + data.invalidSummary + ", Errors: " + data.errors + ". Took " + data.duration + "." + errorSample;
        } catch (err) {
          status.textContent = "Load failed: " + err;
        } finally {
          button.disabled = false;
        }
      });
    </script>
  </body>
</html>
`

type testgridLoadResponse struct {
	Dashboard       string   `json:"dashboard"`
	JobName         string   `json:"jobName"`
	TotalJobs       int      `json:"totalJobs"`
	Inserted        int      `json:"inserted"`
	MissingSummary  int      `json:"missingSummary"`
	InvalidSummary  int      `json:"invalidSummary"`
	Errors          int      `json:"errors"`
	ErrorSamples    []string `json:"errorSamples,omitempty"`
	Duration        string   `json:"duration"`
	MaxPages        int      `json:"maxPages"`
	Concurrency     int      `json:"concurrency"`
}

type testgridLoadStats struct {
	mu             sync.Mutex
	totalJobs      int
	inserted       int
	missingSummary int
	invalidSummary int
	errors         int
	errorSamples   []string
}

func (s *testgridLoadStats) addInserted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inserted++
}

func (s *testgridLoadStats) addMissingSummary() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.missingSummary++
}

func (s *testgridLoadStats) addInvalidSummary(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidSummary++
	s.addErrorSampleLocked(err)
}

func (s *testgridLoadStats) addError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors++
	s.addErrorSampleLocked(err)
}

func (s *testgridLoadStats) addErrorSampleLocked(err error) {
	if len(s.errorSamples) >= maxErrorSampleCount {
		return
	}
	s.errorSamples = append(s.errorSamples, err.Error())
}

func (s *testgridLoadStats) response(duration time.Duration, maxPages, concurrency int) testgridLoadResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return testgridLoadResponse{
		Dashboard:      testgridDashboard,
		JobName:        testgridJobName,
		TotalJobs:      s.totalJobs,
		Inserted:       s.inserted,
		MissingSummary: s.missingSummary,
		InvalidSummary: s.invalidSummary,
		Errors:         s.errors,
		ErrorSamples:   append([]string(nil), s.errorSamples...),
		Duration:       duration.String(),
		MaxPages:       maxPages,
		Concurrency:    concurrency,
	}
}

// LoadTestGrid crawls TestGrid job history and loads gopogh summaries into the DB.
func (m *DB) LoadTestGrid(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(testgridLoaderHTML))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", fmt.Sprintf("%s, %s", http.MethodGet, http.MethodPost))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := m.Database.Initialize(); err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize database: %v", err), http.StatusInternalServerError)
		return
	}

	maxPages := parsePositiveInt(r.URL.Query().Get("max_pages"), defaultMaxPages)
	concurrency := parsePositiveInt(r.URL.Query().Get("concurrency"), defaultConcurrency)
	log.Printf("load-testgrid start dashboard=%s job=%s max_pages=%d concurrency=%d", testgridDashboard, testgridJobName, maxPages, concurrency)

	c := crawler.New(crawler.Config{
		JobName:      testgridJobName,
		MaxPages:     maxPages,
		SkipStatuses: []string{crawler.StatusAborted},
	})
	jobs, err := c.Run()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to crawl testgrid: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("load-testgrid fetched %d jobs", len(jobs))

	stats := &testgridLoadStats{totalJobs: len(jobs)}
	client := &http.Client{Timeout: summaryFetchTimeout}
	start := time.Now()
	ctx := r.Context()

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, job := range jobs {
		job := job
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() {
				<-sem
				wg.Done()
			}()
			if ctx.Err() != nil {
				return
			}
			if err := m.processTestGridJob(ctx, client, job, stats); err != nil {
				stats.addError(err)
			}
		}()
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	resp := stats.response(time.Since(start), maxPages, concurrency)
	log.Printf("load-testgrid finished inserted=%d missing=%d invalid=%d errors=%d duration=%s", resp.Inserted, resp.MissingSummary, resp.InvalidSummary, resp.Errors, resp.Duration)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to write JSON response", http.StatusInternalServerError)
		return
	}
}

func (m *DB) processTestGridJob(ctx context.Context, client *http.Client, job crawler.ProwJob, stats *testgridLoadStats) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	summaryURL, err := spyglassToSummaryURL(job.SpyglassLink)
	if err != nil {
		return fmt.Errorf("job %s: %v", job.ID, err)
	}
	startedAt, err := time.Parse(time.RFC3339, job.Started)
	if err != nil {
		return fmt.Errorf("job %s: failed to parse start time %q: %v", job.ID, job.Started, err)
	}
	summary, err := fetchSummary(ctx, client, summaryURL)
	if err != nil {
		if errors.Is(err, errSummaryNotFound) {
			stats.addMissingSummary()
			return nil
		}
		return fmt.Errorf("job %s: failed to fetch summary: %v", job.ID, err)
	}
	summary.Detail.Details = ensureTestGridDetails(summary.Detail.Details, job.ID)
	if err := summary.Validate(); err != nil {
		stats.addInvalidSummary(fmt.Errorf("job %s: invalid summary: %v", job.ID, err))
		return nil
	}
	dbEnv, dbTests, err := summary.ToDBRows(startedAt)
	if err != nil {
		stats.addInvalidSummary(fmt.Errorf("job %s: invalid summary conversion: %v", job.ID, err))
		return nil
	}
	if err := m.Database.Set(dbEnv, dbTests); err != nil {
		return fmt.Errorf("job %s: failed to insert: %v", job.ID, err)
	}
	stats.addInserted()
	return nil
}

func ensureTestGridDetails(details, jobID string) string {
	details = strings.TrimSpace(details)
	if !strings.HasPrefix(details, "testgrid:") {
		if details == "" {
			details = fmt.Sprintf("testgrid:%s", testgridJobName)
		} else {
			details = fmt.Sprintf("testgrid:%s:%s", testgridJobName, details)
		}
	}
	if jobID == "" {
		return details
	}
	if !strings.HasSuffix(details, jobID) {
		details = details + ":" + jobID
	}
	return details
}

func spyglassToSummaryURL(spyglassLink string) (string, error) {
	if spyglassLink == "" {
		return "", fmt.Errorf("missing spyglass link")
	}
	if strings.HasPrefix(spyglassLink, "/") {
		spyglassLink = "https://prow.k8s.io" + spyglassLink
	}
	if strings.HasPrefix(spyglassLink, "https://storage.googleapis.com/") {
		spyglassLink = strings.TrimSuffix(spyglassLink, "/")
		return spyglassLink + "/artifacts/test_summary.json", nil
	}
	parsed, err := url.Parse(spyglassLink)
	if err != nil {
		return "", fmt.Errorf("invalid spyglass link: %v", err)
	}
	path := strings.TrimPrefix(parsed.Path, "/view/gs/")
	if path == parsed.Path {
		return "", fmt.Errorf("unsupported spyglass path: %s", parsed.Path)
	}
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return "", fmt.Errorf("empty spyglass path")
	}
	return "https://storage.googleapis.com/" + path + "/artifacts/test_summary.json", nil
}

func fetchSummary(ctx context.Context, client *http.Client, summaryURL string) (report.Summary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, summaryURL, nil)
	if err != nil {
		return report.Summary{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return report.Summary{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return report.Summary{}, errSummaryNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return report.Summary{}, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, summaryURL)
	}

	var summary report.Summary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return report.Summary{}, err
	}
	return summary, nil
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
