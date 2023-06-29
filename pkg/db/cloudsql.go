package db

import (
	"fmt"
	"strings"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres" // Blank import used for registering cloudsql driver as a database driver
	"github.com/jmoiron/sqlx"
)

// newCloudSQL returns a new Google Cloud SQL database
func newCloudSQL(cfg Config) (datab, error) {
	return NewCloudPostgres(cfg)
}

// NewCloudPostgres returns a new Google Cloud Postgres database instance
func NewCloudPostgres(cfg Config) (*Postgres, error) {
	cfg.Path = strings.Trim(cfg.Path, " ")
	cfg.Path = strings.Trim(cfg.Path, "'")
	if !strings.Contains(cfg.Path, "sslmode=disable") {
		cfg.Path += " sslmode=disable"
	}
	dbx, err := sqlx.Connect("cloudsqlpostgres", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("cloudsqlpostgres open: %v", err)
	}

	return &Postgres{db: dbx, path: cfg.Path}, nil
}
