//go:build integration

package testutil

import (
	"context"
	"os"
	"path/filepath"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const postgresImage = "postgres:16-alpine"

// StartPostgres starts a Postgres container with the gitgres schema and ref functions.
// Returns connection string (for PGCONN), cleanup function, and error.
// If initScriptPaths is nil or empty, uses testdata/01-04 SQL scripts relative to module root.
func StartPostgres(ctx context.Context, initScriptPaths []string) (connStr string, cleanup func(), err error) {
	if len(initScriptPaths) == 0 {
		root, err := findModuleRoot()
		if err != nil {
			return "", nil, err
		}
		initScriptPaths = []string{
			filepath.Join(root, "testdata", "01_schema.sql"),
			filepath.Join(root, "testdata", "02_object_hash.sql"),
			filepath.Join(root, "testdata", "03_object_read_write.sql"),
			filepath.Join(root, "testdata", "04_ref_functions.sql"),
		}
	}

	ctr, err := postgres.Run(ctx, postgresImage,
		postgres.WithOrderedInitScripts(initScriptPaths...),
		postgres.WithDatabase("gitgres_test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return "", nil, err
	}

	connStr, err = ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return "", nil, err
	}

	cleanup = func() {
		_ = ctr.Terminate(ctx)
	}
	return connStr, cleanup, nil
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
