-- Write/read git objects (copy of gitgres/sql/functions/object_read_write.sql)
CREATE OR REPLACE FUNCTION git_object_write(
    p_repo_id integer,
    p_type smallint,
    p_content bytea
)
RETURNS bytea
LANGUAGE plpgsql AS $$
DECLARE
    v_oid bytea;
BEGIN
    v_oid := git_object_hash(p_type, p_content);
    INSERT INTO objects (repo_id, oid, type, size, content)
    VALUES (p_repo_id, v_oid, p_type, octet_length(p_content), p_content)
    ON CONFLICT (repo_id, oid) DO NOTHING;
    RETURN v_oid;
END;
$$;

CREATE OR REPLACE FUNCTION git_object_read(
    p_repo_id integer,
    p_oid bytea
)
RETURNS TABLE(type smallint, size integer, content bytea)
LANGUAGE sql STABLE STRICT AS $$
    SELECT o.type, o.size, o.content
    FROM objects o
    WHERE o.repo_id = p_repo_id AND o.oid = p_oid;
$$;
