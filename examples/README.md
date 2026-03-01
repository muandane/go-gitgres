# go-gitgres examples: vulnerability scan and SBOM

These examples demonstrate the full pipeline: fetch a repo from a Git URL, store it in Postgres via go-gitgres, clone it back from Postgres, then run vulnerability and SBOM scanning.

## Prerequisites

- Postgres running with the gitgres schema applied (see parent [gitgres](../gitgres/) for `createdb` and schema).
- **Trivy example:** [trivy](https://github.com/aquasecurity/trivy) binary in PATH (`trivy fs`).
- **Syft-Grype example:** [syft](https://github.com/anchore/syft) and [grype](https://github.com/anchore/grype) binaries in PATH.

## Pipeline

1. **Fetch** (optional) — Clone from a Git URL into a temp dir.
2. **Push to Postgres** — Store the repo in Postgres using go-gitgres.
3. **Clone from Postgres** — Clone the repo from Postgres to a temp dir (worktree checked out).
4. **Scan** — Run the scanner on the cloned directory (Trivy or Syft+Grype).

## Commands

Both examples use the same flags:

- `-conn` — Postgres connection string (e.g. `dbname=gitgres_test`).
- `-repo` — Repository name in Postgres.
- `-url` — (Optional) Git URL to fetch. If set, the example fetches from this URL, pushes to Postgres, then clones and scans. If omitted, the repo must already exist in Postgres.

### Trivy (vulnerability scan)

Scans the cloned filesystem with Trivy for vulnerabilities.

```bash
go run ./examples/trivy -conn "dbname=gitgres_test" -repo nginx -url https://github.com/nginx/nginx
```

Without `-url` (repo already in Postgres):

```bash
go run ./examples/trivy -conn "dbname=gitgres_test" -repo nginx
```

### Syft + Grype (SBOM then vulnerability scan)

Generates an SBOM with Syft from the cloned dir, then runs Grype on the SBOM.

```bash
go run ./examples/syft-grype -conn "dbname=gitgres_test" -repo nginx -url https://github.com/nginx/nginx
```

Without `-url`:

```bash
go run ./examples/syft-grype -conn "dbname=gitgres_test" -repo nginx
```

## Layout

- `common/fetch_push.go` — Fetch from Git URL and push to Postgres.
- `common/clone.go` — Clone from Postgres to a temp dir (with worktree checkout).
- `trivy/main.go` — Pipeline + Trivy filesystem scan.
- `syft-grype/main.go` — Pipeline + Syft catalog + Grype on SBOM.
