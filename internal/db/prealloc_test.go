package db

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestListRefsPreallocMatchesListRefs(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_prealloc_refs")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	compareRefs := func(t *testing.T, a, b []ListRefsRow) {
		t.Helper()
		if len(a) != len(b) {
			t.Fatalf("len: got %d, want %d", len(a), len(b))
		}
		for i := range a {
			if a[i].Name != b[i].Name {
				t.Errorf("[%d] Name: got %q, want %q", i, a[i].Name, b[i].Name)
			}
			if !bytes.Equal(a[i].Oid, b[i].Oid) {
				t.Errorf("[%d] Oid: mismatch", i)
			}
			if a[i].Symbolic.Valid != b[i].Symbolic.Valid || a[i].Symbolic.String != b[i].Symbolic.String {
				t.Errorf("[%d] Symbolic: got Valid=%v %q, want Valid=%v %q", i, a[i].Symbolic.Valid, a[i].Symbolic.String, b[i].Symbolic.Valid, b[i].Symbolic.String)
			}
		}
	}

	// 0 refs
	got, err := q.ListRefsPrealloc(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefsPrealloc(0 refs): %v", err)
	}
	want, err := q.ListRefs(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefs(0 refs): %v", err)
	}
	compareRefs(t, got, want)

	// 1 ref
	oid := make([]byte, 20)
	for i := range oid {
		oid[i] = 0x01
	}
	if _, err := q.RefUpdate(ctx, RefUpdateParams{RepoID: repoID, Name: "refs/heads/main", NewOid: oid, OldOid: nil, Force: true}); err != nil {
		t.Fatalf("RefUpdate: %v", err)
	}
	got, err = q.ListRefsPrealloc(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefsPrealloc(1 ref): %v", err)
	}
	want, err = q.ListRefs(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefs(1 ref): %v", err)
	}
	compareRefs(t, got, want)

	// 2 more refs (3 total)
	if err := q.RefInsertSymbolic(ctx, RefInsertSymbolicParams{RepoID: repoID, Name: "HEAD", Symbolic: pgtype.Text{String: "refs/heads/main", Valid: true}}); err != nil {
		t.Fatalf("RefInsertSymbolic: %v", err)
	}
	oid2 := make([]byte, 20)
	for i := range oid2 {
		oid2[i] = 0x02
	}
	if _, err := q.RefUpdate(ctx, RefUpdateParams{RepoID: repoID, Name: "refs/heads/other", NewOid: oid2, OldOid: nil, Force: true}); err != nil {
		t.Fatalf("RefUpdate other: %v", err)
	}
	got, err = q.ListRefsPrealloc(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefsPrealloc(3 refs): %v", err)
	}
	want, err = q.ListRefs(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefs(3 refs): %v", err)
	}
	compareRefs(t, got, want)
}

func TestListObjectOidsPreallocMatchesListObjectOids(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_prealloc_oids")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	compareOids := func(t *testing.T, a, b [][]byte) {
		t.Helper()
		if len(a) != len(b) {
			t.Fatalf("len: got %d, want %d", len(a), len(b))
		}
		for i := range a {
			if !bytes.Equal(a[i], b[i]) {
				t.Errorf("[%d] oid mismatch", i)
			}
		}
	}

	// 0 objects
	got, err := q.ListObjectOidsPrealloc(ctx, repoID)
	if err != nil {
		t.Fatalf("ListObjectOidsPrealloc(0): %v", err)
	}
	want, err := q.ListObjectOids(ctx, repoID)
	if err != nil {
		t.Fatalf("ListObjectOids(0): %v", err)
	}
	compareOids(t, got, want)

	// 1 object
	_, err = q.ObjectWrite(ctx, ObjectWriteParams{RepoID: repoID, ObjType: 3, Content: []byte("one")})
	if err != nil {
		t.Fatalf("ObjectWrite: %v", err)
	}
	got, err = q.ListObjectOidsPrealloc(ctx, repoID)
	if err != nil {
		t.Fatalf("ListObjectOidsPrealloc(1): %v", err)
	}
	want, err = q.ListObjectOids(ctx, repoID)
	if err != nil {
		t.Fatalf("ListObjectOids(1): %v", err)
	}
	compareOids(t, got, want)

	// 2 more (3 total)
	_, err = q.ObjectWrite(ctx, ObjectWriteParams{RepoID: repoID, ObjType: 3, Content: []byte("two")})
	if err != nil {
		t.Fatalf("ObjectWrite 2: %v", err)
	}
	_, err = q.ObjectWrite(ctx, ObjectWriteParams{RepoID: repoID, ObjType: 3, Content: []byte("three")})
	if err != nil {
		t.Fatalf("ObjectWrite 3: %v", err)
	}
	got, err = q.ListObjectOidsPrealloc(ctx, repoID)
	if err != nil {
		t.Fatalf("ListObjectOidsPrealloc(3): %v", err)
	}
	want, err = q.ListObjectOids(ctx, repoID)
	if err != nil {
		t.Fatalf("ListObjectOids(3): %v", err)
	}
	compareOids(t, got, want)
}
