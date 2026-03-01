-- name: ObjectWrite :one
SELECT git_object_write(sqlc.arg(repo_id)::integer, sqlc.arg(obj_type)::smallint, sqlc.arg(content)) AS oid;

-- name: ObjectRead :one
SELECT type, size, content FROM objects WHERE repo_id = $1 AND oid = $2;

-- name: ListObjectOids :many
SELECT oid FROM objects WHERE repo_id = $1;
