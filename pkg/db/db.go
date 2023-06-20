package db

import (
	"fmt"
	"os"

	"github.com/medyagh/gopogh/pkg/models"
)

// Config is database configuration
type Config struct {
	Type string
	Path string
}

// datab is the database interface we support
type datab interface {
	Set(models.DBEnvironmentTest, []models.DBTestCase) error

	Initialize() error
}

// New handles which database driver to use and initializes the db
func New(cfg Config) (datab, error) {
	switch cfg.Type {
	case "sqlite":
		return NewSQLite(cfg)
	default:
		return nil, fmt.Errorf("unknown backend: %q", cfg.Type)
	}
}

// FromEnv configures and returns a database instance.
// backend and path parameters are default config, otherwise gets config from the environment variables DB_BACKEND and DB_PATH
func FromEnv(backend string, path string) (datab, error) {
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

	c, err := New(Config{
		Type: backend,
		Path: path,
	})
	if err != nil {
		return nil, fmt.Errorf("new from %s: %s: %w", backend, path, err)
	}

	return c, nil
}
