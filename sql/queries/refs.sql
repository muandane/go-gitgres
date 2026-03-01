-- name: RefUpdate :one
SELECT git_ref_update(
    sqlc.arg(repo_id)::integer,
    sqlc.arg(name)::text,
    sqlc.arg(new_oid),
    sqlc.arg(old_oid),
    sqlc.arg(force)::boolean
) AS ok;

-- name: RefSetSymbolic :exec
SELECT git_ref_set_symbolic(sqlc.arg(repo_id)::integer, sqlc.arg(name)::text, sqlc.arg(target)::text);

-- name: ListRefsCount :one
SELECT count(*)::int FROM refs WHERE repo_id = $1;

-- name: ListRefs :many
SELECT name, oid, symbolic FROM refs WHERE repo_id = $1 ORDER BY name;

-- Remote-helper push: direct ref write/delete (no reflog, no CAS)

-- name: RefInsert :exec
INSERT INTO refs (repo_id, name, oid)
VALUES ($1, $2, $3)
ON CONFLICT (repo_id, name) DO UPDATE SET oid = $3, symbolic = NULL;

-- name: RefInsertSymbolic :exec
INSERT INTO refs (repo_id, name, symbolic)
VALUES ($1, $2, $3)
ON CONFLICT (repo_id, name) DO UPDATE SET oid = NULL, symbolic = $3;

-- name: RefDelete :exec
DELETE FROM refs WHERE repo_id = $1 AND name = $2;

-- name: RefsHasHEAD :one
SELECT 1 FROM refs WHERE repo_id = $1 AND name = 'HEAD' ORDER BY repo_id LIMIT 1;
