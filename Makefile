.PHONY: all build test test-coverage test-integration clean sqlc lint-sql

all: build

build:
	go build -o gitgres-backend ./cmd/backend
	ln -sf gitgres-backend git-remote-gitgres

sqlc:
	sqlc generate

test:
	go test ./...

# Test coverage (unit + DB tests that run when PGCONN is set). Low %% is expected without a DB.
# Writes coverage.out; view HTML: go tool cover -html=coverage.out
test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

# Full coverage: integration tests run against Postgres in Docker (requires Docker).
# Use this for near-100%% coverage; test-coverage reflects only tests that run without a DB.
test-integration:
	go test -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

# Lint SQL with sqlfluff (pip install sqlfluff).
lint-sql:
	sqlfluff lint sql/

clean:
	rm -f gitgres-backend git-remote-gitgres coverage.out
