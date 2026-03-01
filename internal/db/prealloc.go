package db

import (
	"context"
)

// ListRefsPrealloc returns refs for the repo. Semantically equivalent to ListRefs.
func (q *Queries) ListRefsPrealloc(ctx context.Context, repoID int32) ([]ListRefsRow, error) {
	count, err := q.ListRefsCount(ctx, repoID)
	if err != nil {
		return nil, err
	}
	items := make([]ListRefsRow, 0, count)
	rows, err := q.db.Query(ctx, listRefs, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var i ListRefsRow
		if err := rows.Scan(&i.Name, &i.Oid, &i.Symbolic); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ListObjectOidsPrealloc returns object OIDs for the repo. Semantically equivalent to ListObjectOids.
func (q *Queries) ListObjectOidsPrealloc(ctx context.Context, repoID int32) ([][]byte, error) {
	count, err := q.ListObjectOidsCount(ctx, repoID)
	if err != nil {
		return nil, err
	}
	items := make([][]byte, 0, count)
	rows, err := q.db.Query(ctx, listObjectOids, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var oid []byte
		if err := rows.Scan(&oid); err != nil {
			return nil, err
		}
		items = append(items, oid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
