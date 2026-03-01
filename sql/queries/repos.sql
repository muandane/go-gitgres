-- name: GetOrCreateRepo :one
INSERT INTO repositories (name)
VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = $1
RETURNING id;

-- name: GetRepo :one
SELECT id FROM repositories WHERE name = $1;
