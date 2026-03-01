package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenPoolForTest opens a pool and pings; skips the test when DB is unavailable.
func OpenPoolForTest(ctx context.Context, t *testing.T, connString string) *pgxpool.Pool {
	t.Helper()
	pool, err := OpenPool(ctx, connString)
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("DB unreachable: %v", err)
	}
	return pool
}
