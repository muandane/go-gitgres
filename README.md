# go-gitgres

Go rewrite of [gitgres](../gitgres/): store Git objects and refs in PostgreSQL. Uses [Go 1.26](https://go.dev/doc/go1.26), [go-git](https://github.com/go-git/go-git), [sqlc](https://sqlc.dev/), and [pgx](https://github.com/jackc/pgx). SQL is linted with [sqlfluff](https://docs.sqlfluff.org/).

## Build

```bash
make build
# Produces gitgres-backend; git-remote-gitgres is a symlink to it.
# Or: go build -o gitgres-backend ./cmd/backend
```

## Usage

One binary for both CLI and Git remote-helper. Same as the C gitgres backend:

```bash
# CLI (schema must be applied via gitgres: make -C ../gitgres createdb)
./gitgres-backend init "dbname=gitgres_test" myrepo
./gitgres-backend push "dbname=gitgres_test" myrepo /path/to/repo
./gitgres-backend clone "dbname=gitgres_test" myrepo /path/to/dest
./gitgres-backend ls-refs "dbname=gitgres_test" myrepo
```

Remote helper: install the binary as `git-remote-gitgres` in PATH (make build creates a symlink). Then:

```bash
git remote add pg gitgres::dbname=gitgres_test/myrepo
git push pg main
git clone gitgres::dbname=gitgres_test/myrepo /path/to/clone
```

## Tests

Pure Go test suite. No Ruby.

```bash
make test
# or
go test ./...

# Coverage: DB tests skip when no Postgres; low %% is expected without a DB
make test-coverage
# HTML report: go tool cover -html=coverage.out

# Full coverage: run DB tests against Postgres in Docker (requires Docker)
make test-integration
```

Tests that hit the DB skip when Postgres is unavailable. So `make test-coverage` without a DB reports only unit-test coverage (low percentage); use `make test-integration` for coverage that includes all DB-backed code. For full coverage without a pre-created DB, run `make test-integration` (uses testcontainers; requires Docker). With a running DB and schema applied (`make -C ../gitgres createdb` once), `make test` runs the same tests against `gitgres_test`.

## Library

Import the storer to use Git-over-Postgres from Go:

```go
import "go-gitgres/internal/db"
import "go-gitgres/internal/storer"

pool, _ := db.OpenPool(ctx, "dbname=gitgres_test")
s, _ := storer.NewPostgresStorer(ctx, pool, "my-repo")
// s implements storage.Storer; use with go-git
```

## SQL codegen and lint

Schema and queries are in `sql/`. Generate Go and lint SQL:

```bash
make sqlc        # sqlc generate
make lint-sql    # sqlfluff lint sql/ (requires: pip install sqlfluff)
```
