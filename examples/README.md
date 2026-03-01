# go-gitgres examples: vulnerability scan and SBOM

These examples demonstrate the full pipeline: fetch a repo from a Git URL, store it in Postgres via go-gitgres, clone it back from Postgres, then run vulnerability and SBOM scanning.

Examples live in a **separate Go module** (`go.mod` in this directory) so their dependencies do not affect the main go-gitgres project. Run them from this directory:

```bash
cd examples
go run ./trivy -conn "..." -repo nginx -url https://github.com/nginx/nginx
go run ./syft-grype -conn "..." -repo nginx -url https://github.com/nginx/nginx
```

## Prerequisites

- Postgres running with the gitgres schema applied (see parent [gitgres](../gitgres/) for `createdb` and schema).
- **Trivy example:** [trivy](https://github.com/aquasecurity/trivy) binary in PATH (`trivy fs`).
- **Syft-Grype example:** [syft](https://github.com/anchore/syft) and [grype](https://github.com/anchore/grype) binaries in PATH.

**Secured Postgres**

Use a connection string that includes TLS and auth; pass it via `-conn`. Prefer env vars for secrets (e.g. `PGPASSWORD`) instead of putting them in the string.

Example:  
`go run ./trivy -conn "host=db.example.com port=5432 dbname=gitgres user=gitgres sslmode=require" -repo nginx -url https://github.com/nginx/nginx`

## Pipeline

1. **Fetch** (optional) — Clone from a Git URL into a temp dir.
2. **Push to Postgres** — Store the repo in Postgres using go-gitgres.
3. **Clone from Postgres** — Clone the repo from Postgres to a temp dir (worktree checked out).
4. **Scan** — Run the scanner on the cloned directory (Trivy or Syft+Grype).

## Commands

Both examples use the same flags:

- `-conn` — Postgres connection string (e.g. `dbname=gitgres_test`; for TLS use `sslmode=require` etc.).
- `-repo` — Repository name in Postgres.
- `-url` — (Optional) Git URL to fetch. If set, the example fetches from this URL, pushes to Postgres, then clones and scans. If omitted, the repo must already exist in Postgres.

### Trivy (vulnerability scan)

Scans the cloned filesystem with Trivy for vulnerabilities. Run from the `examples/` directory:

```bash
cd examples
go run ./trivy -conn "dbname=gitgres_test" -repo nginx -url https://github.com/nginx/nginx
```

Without `-url` (repo already in Postgres):

```bash
go run ./trivy -conn "dbname=gitgres_test" -repo nginx
```

### Syft + Grype (SBOM then vulnerability scan)

Generates an SBOM with Syft from the cloned dir, then runs Grype on the SBOM. Run from the `examples/` directory:

```bash
cd examples
go run ./syft-grype -conn "dbname=gitgres_test" -repo nginx -url https://github.com/nginx/nginx
```

Without `-url`:

```bash
go run ./syft-grype -conn "dbname=gitgres_test" -repo nginx
```

## Layout

- `go.mod` — Separate module; only depends on `go-gitgres` (no extra deps in main project).
- `trivy/main.go` — Pipeline + Trivy filesystem scan (uses `go-gitgres/pkg/clone`).
- `syft-grype/main.go` — Pipeline + Syft catalog + Grype on SBOM (uses `go-gitgres/pkg/clone`).
