package db

import (
	"fmt"
	"os"

	"github.com/medyagh/gopogh/pkg/models"
)

// config is database configuration
type config struct {
	Type string
	Path string
}

// datab is the database interface we support
type datab interface {
	Set(models.DBEnvironmentTest, []models.DBTestCase) error

	Initialize() error
}

// newDB handles which database driver to use and initializes the db
func newDB(cfg config) (datab, error) {
	switch cfg.Type {
	case "sqlite":
		return newSQLite(cfg)
	case "postgres":
		return newPostgres(cfg)
	case "cloudsql":
		return newCloudSQL(cfg)
	default:
		return nil, fmt.Errorf("unknown backend: %q", cfg.Type)
	}
}

// FromEnv configures and returns a database instance.
// backend and path parameters are default config, otherwise gets config from the environment variables DB_BACKEND and DB_PATH
func FromEnv(path string, backend string) (datab, error) {
	if backend == "" {
		backend = os.Getenv("DB_BACKEND")
	}
	if backend == "" {
		return nil, fmt.Errorf("missing DB_BACKEND")
	}

	if path == "" {
		path = os.Getenv("DB_PATH")
	}
	if path == "" {
		return nil, fmt.Errorf("missing DB_PATH")
	}

	c, err := newDB(config{
		Type: backend,
		Path: path,
	})
	if err != nil {
		return nil, fmt.Errorf("new from %s: %s: %v", backend, path, err)
	}

	return c, nil
}
