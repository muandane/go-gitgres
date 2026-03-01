# go-gitgres

Go rewrite of [gitgres](../gitgres/): store Git objects and refs in PostgreSQL. Uses [Go 1.26](https://go.dev/doc/go1.26), [go-git](https://github.com/go-git/go-git), [sqlc](https://sqlc.dev/), and [pgx](https://github.com/jackc/pgx). SQL is linted with [sqlfluff](https://docs.sqlfluff.org/).

## Build

```bash
make build
# or
go build -o gitgres-backend ./cmd/backend
go build -o git-remote-gitgres ./cmd/remote-helper
```

## Usage

Same as the C gitgres backend:

```bash
# Create repo in DB (schema must be applied via gitgres: make -C ../gitgres createdb)
./gitgres-backend init "dbname=gitgres_test" myrepo
./gitgres-backend push "dbname=gitgres_test" myrepo /path/to/repo
./gitgres-backend clone "dbname=gitgres_test" myrepo /path/to/dest
./gitgres-backend ls-refs "dbname=gitgres_test" myrepo
```

Remote helper (for `git push` / `git clone`):

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

# Coverage
make test-coverage
# HTML report: go tool cover -html=coverage.out
```

Tests that hit the DB skip when Postgres is unavailable. With a running DB and schema applied (`make -C ../gitgres createdb` once), the same tests run against `gitgres_test`.

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
