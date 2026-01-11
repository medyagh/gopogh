package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// TestGridDashboard describes a single TestGrid dashboard entry.
type TestGridDashboard struct {
	ID           string   `json:"id"`
	JobName      string   `json:"job_name"`
	Label        string   `json:"label"`
	SkipStatuses []string `json:"skip_statuses"`
	MinDuration  string   `json:"min_duration"`
	MaxPages     int      `json:"max_pages"`
}

// TestGridConfig holds multiple TestGrid dashboard entries.
type TestGridConfig struct {
	Dashboards []TestGridDashboard `json:"dashboards"`
}

// DefaultTestGridConfig provides a fallback dashboard list.
func DefaultTestGridConfig() TestGridConfig {
	return TestGridConfig{
		Dashboards: []TestGridDashboard{
			{
				ID:           "minikube-periodics#ci-minikube-integration",
				JobName:      "ci-minikube-integration",
				Label:        "minikube-periodics#ci-minikube-integration",
				SkipStatuses: []string{"ABORTED"},
				MaxPages:     defaultMaxPages,
			},
		},
	}
}

// LoadTestGridConfig loads TestGrid dashboards from a JSON file.
func LoadTestGridConfig(path string) (TestGridConfig, error) {
	if path == "" {
		return DefaultTestGridConfig(), fmt.Errorf("missing testgrid config path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultTestGridConfig(), err
	}
	var cfg TestGridConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultTestGridConfig(), err
	}
	cfg = cfg.normalize()
	if len(cfg.Dashboards) == 0 {
		return DefaultTestGridConfig(), fmt.Errorf("no dashboards configured")
	}
	return cfg, nil
}

// FindDashboard locates a dashboard by id or job name.
func (c TestGridConfig) FindDashboard(key string) (TestGridDashboard, bool) {
	if key == "" {
		return TestGridDashboard{}, false
	}
	for _, d := range c.Dashboards {
		if d.ID == key || d.JobName == key {
			return d, true
		}
	}
	return TestGridDashboard{}, false
}

func (c TestGridConfig) normalize() TestGridConfig {
	if len(c.Dashboards) == 0 {
		return c
	}
	out := make([]TestGridDashboard, 0, len(c.Dashboards))
	for _, d := range c.Dashboards {
		normalized := normalizeDashboard(d)
		if normalized.ID == "" && normalized.JobName == "" {
			continue
		}
		if normalized.ID == "" {
			normalized.ID = normalized.JobName
		}
		if normalized.Label == "" {
			normalized.Label = normalized.ID
		}
		out = append(out, normalized)
	}
	return TestGridConfig{Dashboards: out}
}

// ParseMinDuration parses the MinDuration field into a time.Duration.
func (d TestGridDashboard) ParseMinDuration() (time.Duration, error) {
	if strings.TrimSpace(d.MinDuration) == "" {
		return 0, nil
	}
	return time.ParseDuration(strings.TrimSpace(d.MinDuration))
}

func normalizeDashboard(d TestGridDashboard) TestGridDashboard {
	d.ID = strings.TrimSpace(d.ID)
	d.JobName = strings.TrimSpace(d.JobName)
	d.Label = strings.TrimSpace(d.Label)
	if len(d.SkipStatuses) > 0 {
		d.SkipStatuses = normalizeStatuses(d.SkipStatuses)
	}
	if d.MaxPages <= 0 {
		d.MaxPages = defaultMaxPages
	}
	if d.JobName == "" && d.ID != "" {
		parts := strings.SplitN(d.ID, "#", 2)
		if len(parts) == 2 && parts[1] != "" {
			d.JobName = strings.TrimSpace(parts[1])
		} else {
			d.JobName = d.ID
		}
	}
	return d
}

func normalizeStatuses(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToUpper(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if normalized == "ABORT" {
			normalized = "ABORTED"
		}
		out = append(out, normalized)
	}
	return out
}
