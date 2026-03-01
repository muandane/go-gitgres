package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenPool opens a connection pool from connString. If empty, uses
// PG connection env (e.g. PGDATABASE=gitgres_test or PGCONN).
func OpenPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	if connString == "" {
		connString = os.Getenv("PGCONN")
		if connString == "" {
			db := os.Getenv("PGDATABASE")
			if db == "" {
				db = "gitgres_test"
			}
			connString = "dbname=" + db
		}
	}
	return pgxpool.New(ctx, connString)
}
