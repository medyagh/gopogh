// Package db provides database integration for gopogh
package db

import (
	"context"
	"fmt"
	"net"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres" // Blank import used for registering cloudsql driver as a database driver
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

// NewCloudSQL returns a new Google Cloud SQL database
func NewCloudSQL(cfg config) (Datab, error) {
	switch cfg.dbType {
	case "postgres":
		return newCloudPostgres(cfg)
	default:
		return nil, fmt.Errorf("unknown cloudsql backend: %q", cfg.dbType)
	}
}

// newCloudPostgres returns a new Google Cloud Postgres database instance
func newCloudPostgres(cfg config) (*Postgres, error) {
	cfg.path = strings.Trim(cfg.path, " '")
	var dbx *sqlx.DB
	var err error
	if cfg.useIAMAuth {
		dbx, err = iamAuth(cfg)
	} else {
		dbx, err = userPassAuth(cfg)
	}

	return &Postgres{db: dbx, path: cfg.path}, err
}

func userPassAuth(cfg config) (*sqlx.DB, error) {
	path := fmt.Sprintf("host=%s %s", cfg.host, cfg.path)
	if !strings.Contains(path, "sslmode=disable") {
		path += " sslmode=disable"
	}
	dbx, err := sqlx.Connect("cloudsqlpostgres", path)
	if err != nil {
		return nil, fmt.Errorf("cloudsqlpostgres open: %v", err)
	}
	return dbx, nil
}

func iamAuth(cfg config) (*sqlx.DB, error) {
	d, err := cloudsqlconn.NewDialer(context.Background(), cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return nil, fmt.Errorf("cloudsqlconn.NewDialer: %v", err)
	}
	config, err := pgx.ParseConfig(cfg.path)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %v", err)
	}
	config.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return d.Dial(ctx, cfg.host)
	}
	dbURI := stdlib.RegisterConnConfig(config)
	dbx, err := sqlx.Open("pgx", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sqlx.Open: %v", err)
	}
	return dbx, nil
}
