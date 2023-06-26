package db

import (
	"fmt"
	"strings"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres" // Blank import used for registering cloudsql driver as a database driver
	"github.com/jmoiron/sqlx"
)

// newCloudSQL returns a new Google Cloud SQL database
func newCloudSQL(cfg config) (datab, error) {
	return newCloudPostgres(cfg)
}

// newCloudPostgres returns a new Google Cloud Postgres database instance
func newCloudPostgres(cfg config) (*Postgres, error) {
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
