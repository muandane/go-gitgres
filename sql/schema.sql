-- Minimal schema for sqlc codegen; matches gitgres/sql/schema.sql.
-- Full DDL + functions are applied by gitgres (make createdb / install-sql).

CREATE TABLE repositories (
    id          serial PRIMARY KEY,
    name        text NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE objects (
    repo_id     integer NOT NULL REFERENCES repositories(id),
    oid         bytea NOT NULL,
    type        smallint NOT NULL,
    size        integer NOT NULL,
    content     bytea NOT NULL,
    PRIMARY KEY (repo_id, oid)
);

CREATE TABLE refs (
    repo_id     integer NOT NULL REFERENCES repositories(id),
    name        text NOT NULL,
    oid         bytea,
    symbolic    text,
    PRIMARY KEY (repo_id, name),
    CHECK ((oid IS NOT NULL) != (symbolic IS NOT NULL))
);
