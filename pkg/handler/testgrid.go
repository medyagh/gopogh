package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
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
	defaultMaxPages = 20
	// defaultConcurrency limits the number of parallel summary fetches when loading TestGrid.
	defaultConcurrency  = 6
	maxErrorSampleCount = 10
	summaryFetchTimeout = 20 * time.Second
)

var errSummaryNotFound = errors.New("summary not found")

var testgridLoaderTemplate = template.Must(template.New("load-testgrid").Funcs(template.FuncMap{
	"eq": func(a, b string) bool { return a == b },
}).Parse(`<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Load TestGrid</title>
    <style>
      body { font-family: Arial, sans-serif; margin: 2rem; }
      button { padding: 0.5rem 1rem; }
      #status { margin-left: 1rem; }
      select { margin-right: 0.5rem; }
    </style>
  </head>
  <body>
    <h2>Load TestGrid Data</h2>
    {{if .HasDashboards}}
    <label for="dashboardSelect">Dashboard:</label>
    <select id="dashboardSelect">
      {{range .Dashboards}}
      <option value="{{.ID}}"{{if eq .ID $.SelectedID}} selected{{end}}>{{.Label}}</option>
      {{end}}
    </select>
    <button id="loadBtn">Load TestGrid</button>
    <span id="status"></span>
    {{else}}
    <p>No dashboards configured.</p>
    {{end}}
    <p><a href="/">Back to charts</a></p>
    <script>
      const button = document.getElementById("loadBtn");
      const status = document.getElementById("status");
      const select = document.getElementById("dashboardSelect");
      if (button && select) {
        button.addEventListener("click", async () => {
          button.disabled = true;
          status.textContent = "Loading...";
          try {
            const dashboard = select.value;
            const response = await fetch("/load-testgrid?dashboard=" + encodeURIComponent(dashboard), { method: "POST" });
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
      }
    </script>
  </body>
</html>
`))

type testgridLoadResponse struct {
	Dashboard      string   `json:"dashboard"`
	JobName        string   `json:"jobName"`
	TotalJobs      int      `json:"totalJobs"`
	Inserted       int      `json:"inserted"`
	MissingSummary int      `json:"missingSummary"`
	InvalidSummary int      `json:"invalidSummary"`
	Errors         int      `json:"errors"`
	ErrorSamples   []string `json:"errorSamples,omitempty"`
	Duration       string   `json:"duration"`
	MaxPages       int      `json:"maxPages"`
	Concurrency    int      `json:"concurrency"`
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

func (s *testgridLoadStats) response(dashboardID, jobName string, duration time.Duration, maxPages, concurrency int) testgridLoadResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return testgridLoadResponse{
		Dashboard:      dashboardID,
		JobName:        jobName,
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
	start := time.Now()
	defer func() {
		log.Printf("duration metric: took %f seconds to initialize Postgres tables\n", time.Since(start).Seconds())
	}()
	cfg := m.TestGridCfg
	if len(cfg.Dashboards) == 0 {
		cfg = DefaultTestGridConfig()
	}
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body, err := renderTestGridLoaderHTML(cfg, r.URL.Query().Get("dashboard"))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to render page: %v", err), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(body))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", fmt.Sprintf("%s, %s", http.MethodGet, http.MethodPost))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(cfg.Dashboards) == 0 {
		http.Error(w, "no testgrid dashboards configured", http.StatusInternalServerError)
		return
	}

	maxPages := parsePositiveInt(r.URL.Query().Get("max_pages"), defaultMaxPages)
	concurrency := parsePositiveInt(r.URL.Query().Get("concurrency"), defaultConcurrency)
	dashboardKey := r.URL.Query().Get("dashboard")
	dashboard, ok := cfg.FindDashboard(dashboardKey)
	if !ok {
		dashboard = cfg.Dashboards[0]
		if dashboardKey != "" {
			http.Error(w, fmt.Sprintf("unknown dashboard %q", dashboardKey), http.StatusBadRequest)
			return
		}
	}
	if dashboard.JobName == "" {
		http.Error(w, "selected dashboard missing job_name", http.StatusBadRequest)
		return
	}
	envGroup := strings.TrimSpace(dashboard.EnvGroup)
	if envGroup == "" {
		envGroup = "Legacy"
	}
	skipStatuses := dashboard.SkipStatuses
	if len(skipStatuses) == 0 {
		skipStatuses = []string{crawler.StatusAborted}
	}
	minDuration, err := dashboard.ParseMinDuration()
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid min_duration for %s: %v", dashboard.ID, err), http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("max_pages") == "" && dashboard.MaxPages > 0 {
		maxPages = dashboard.MaxPages
	}
	log.Printf("load-testgrid start dashboard=%s job=%s max_pages=%d concurrency=%d min_duration=%s skip_statuses=%v", dashboard.ID, dashboard.JobName, maxPages, concurrency, minDuration.String(), skipStatuses)

	c := crawler.New(crawler.Config{
		JobName:      dashboard.JobName,
		MaxPages:     maxPages,
		SkipStatuses: skipStatuses,
		MinDuration:  minDuration,
	})
	jobs, err := c.Run()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to crawl testgrid: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("load-testgrid fetched %d jobs for dashboard=%s", len(jobs), dashboard.ID)

	stats := &testgridLoadStats{totalJobs: len(jobs)}
	client := &http.Client{Timeout: summaryFetchTimeout}
	start2 := time.Now()
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
			if err := m.processTestGridJob(ctx, client, job, dashboard.JobName, envGroup, stats); err != nil {
				stats.addError(err)
			}
		}()
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	resp := stats.response(dashboard.ID, dashboard.JobName, time.Since(start2), maxPages, concurrency)
	log.Printf("load-testgrid finished inserted=%d missing=%d invalid=%d errors=%d duration=%s", resp.Inserted, resp.MissingSummary, resp.InvalidSummary, resp.Errors, resp.Duration)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to write JSON response", http.StatusInternalServerError)
		return
	}
}

func (m *DB) processTestGridJob(ctx context.Context, client *http.Client, job crawler.ProwJob, jobName, envGroup string, stats *testgridLoadStats) error {
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
	artifactPath, err := summaryURLToArtifactPath(summaryURL)
	if err != nil {
		log.Printf("job %s: failed to parse artifact path: %v", job.ID, err)
	}
	summary.Detail.Details = ensureTestGridDetails(summary.Detail.Details, jobName, job.ID)
	if err := summary.Validate(); err != nil {
		stats.addInvalidSummary(fmt.Errorf("job %s: invalid summary: %v", job.ID, err))
		return nil
	}
	dbEnv, dbTests, err := summary.ToDBRows(startedAt)
	if err != nil {
		stats.addInvalidSummary(fmt.Errorf("job %s: invalid summary conversion: %v", job.ID, err))
		return nil
	}
	dbEnv.EnvGroup = envGroup
	if artifactPath != "" {
		dbEnv.ArtifactPath = artifactPath
	}
	if err := m.Database.Set(dbEnv, dbTests); err != nil {
		return fmt.Errorf("job %s: failed to insert: %v", job.ID, err)
	}
	stats.addInserted()
	return nil
}

func ensureTestGridDetails(details, jobName, jobID string) string {
	details = strings.TrimSpace(details)
	if strings.HasPrefix(strings.ToLower(details), "testgrid:") {
		details = strings.TrimSpace(details[len("testgrid:"):])
	}
	if jobName != "" {
		prefix := jobName + ":"
		if strings.HasPrefix(details, prefix) {
			details = strings.TrimSpace(details[len(prefix):])
		}
	}
	if details == "" {
		return jobID
	}
	if jobID == "" {
		return details
	}
	for _, token := range strings.Split(details, ":") {
		if strings.TrimSpace(token) == jobID {
			return details
		}
	}
	return details + ":" + jobID
}

type testgridLoaderPageData struct {
	Dashboards    []TestGridDashboard
	SelectedID    string
	HasDashboards bool
}

func renderTestGridLoaderHTML(cfg TestGridConfig, selectedID string) (string, error) {
	if len(cfg.Dashboards) == 0 {
		return "", fmt.Errorf("no dashboards configured")
	}
	if selectedID == "" {
		selectedID = cfg.Dashboards[0].ID
	}
	data := testgridLoaderPageData{
		Dashboards:    cfg.Dashboards,
		SelectedID:    selectedID,
		HasDashboards: len(cfg.Dashboards) > 0,
	}
	var buf bytes.Buffer
	if err := testgridLoaderTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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

func summaryURLToArtifactPath(summaryURL string) (string, error) {
	parsed, err := url.Parse(summaryURL)
	if err != nil {
		return "", fmt.Errorf("invalid summary url: %v", err)
	}
	if !strings.HasSuffix(parsed.Path, "/artifacts/test_summary.json") {
		return "", fmt.Errorf("unexpected summary url path: %s", parsed.Path)
	}
	path := strings.TrimPrefix(parsed.Path, "/")
	path = strings.TrimSuffix(path, "/artifacts/test_summary.json")
	path = strings.TrimPrefix(path, "gs/")
	if path == "" {
		return "", fmt.Errorf("unable to derive artifact path from %q", summaryURL)
	}
	return path, nil
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
