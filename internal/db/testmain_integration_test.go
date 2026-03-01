//go:build integration

package db

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/muandane/go-gitgres/internal/testutil"
)

func TestMain(m *testing.M) {
	if os.Getenv("PGCONN") != "" {
		os.Exit(m.Run())
		return
	}
	ctx := context.Background()
	connStr, cleanup, err := testutil.StartPostgres(ctx, nil)
	if err != nil {
		log.Fatalf("testcontainers: %v", err)
	}
	defer cleanup()
	os.Setenv("PGCONN", connStr)
	os.Exit(m.Run())
}
