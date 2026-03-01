# go-gitgres

Store Git objects and refs in PostgreSQL. One binary acts as CLI and Git remote-helper.

Uses Go 1.26, [go-git](https://github.com/go-git/go-git), [sqlc](https://sqlc.dev/), and [pgx](https://github.com/jackc/pgx).

## Build

```bash
make build
```

Produces `gitgres-backend` and a `git-remote-gitgres` symlink. Or: `go build -o gitgres-backend ./cmd/backend`.

## Usage

Apply the schema first (see parent [gitgres](../gitgres/) for `createdb`). Then:

**CLI**

```bash
./gitgres-backend init "dbname=gitgres_test" myrepo
./gitgres-backend push "dbname=gitgres_test" myrepo /path/to/repo
./gitgres-backend clone "dbname=gitgres_test" myrepo /path/to/dest
./gitgres-backend ls-refs "dbname=gitgres_test" myrepo
```

**Remote helper**

Install the binary as `git-remote-gitgres` in PATH. Then:

```bash
git remote add pg gitgres::dbname=gitgres_test/myrepo
git push pg main
git clone gitgres::dbname=gitgres_test/myrepo /path/to/clone
```

## Tests

```bash
make test
# or
go test ./...
```

DB tests skip when Postgres is unavailable. For coverage:

```bash
make test-coverage
# HTML: go tool cover -html=coverage.out
```

For full coverage including DB-backed code (uses testcontainers; requires Docker):

```bash
make test-integration
```

## Library

Use the storer from Go. See `go doc go-gitgres/internal/db`, `go doc go-gitgres/internal/storer`, `go doc go-gitgres/internal/backend`.

```go
import "go-gitgres/internal/db"
import "go-gitgres/internal/storer"

pool, _ := db.OpenPool(ctx, "dbname=gitgres_test")
s, _ := storer.NewPostgresStorer(ctx, pool, "my-repo")
// s implements storage.Storer for go-git
```

## SQL

Schema and queries live in `sql/`. Generate Go and lint:

```bash
make sqlc       # sqlc generate
make lint-sql   # sqlfluff lint sql/ (pip install sqlfluff)
```
