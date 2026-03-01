.PHONY: all build test test-coverage test-integration clean sqlc lint-sql

all: build

build:
	go build -o gitgres-backend ./cmd/backend
	ln -sf gitgres-backend git-remote-gitgres

sqlc:
	sqlc generate

test:
	go test ./...

# Test coverage: writes coverage.out, prints summary. View HTML: go tool cover -html=coverage.out
test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

# Near-100%% coverage: runs DB tests against Postgres in Docker (requires Docker). Same tests, no skip.
test-integration:
	go test -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

# Lint SQL with sqlfluff (pip install sqlfluff).
lint-sql:
	sqlfluff lint sql/

clean:
	rm -f gitgres-backend git-remote-gitgres coverage.out
