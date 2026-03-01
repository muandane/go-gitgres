package main

import (
	"context"
	"os"
	"testing"

	"go-gitgres/internal/db"
)

func TestRunInit(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	connStr := os.Getenv("PGCONN")
	if connStr == "" {
		connStr = "dbname=" + os.Getenv("PGDATABASE")
		if connStr == "dbname=" {
			connStr = "dbname=gitgres_test"
		}
	}

	runInit(ctx, connStr, "go_backend_init_test")
	// Idempotent
	runInit(ctx, connStr, "go_backend_init_test")
}

func TestRunLsRefs(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	connStr := os.Getenv("PGCONN")
	if connStr == "" {
		connStr = "dbname=gitgres_test"
	}

	// Ensure repo exists
	runInit(ctx, connStr, "go_backend_lsrefs_test")
	runLsRefs(ctx, connStr, "go_backend_lsrefs_test")
}
