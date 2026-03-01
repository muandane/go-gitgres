package db

import (
	"context"
	"testing"
)

func TestObjectWriteRead(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_obj_repo")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	content := []byte("hello world")
	oidRes, err := q.ObjectWrite(ctx, ObjectWriteParams{
		RepoID:  repoID,
		ObjType: 3, // blob
		Content: content,
	})
	if err != nil {
		t.Fatalf("ObjectWrite: %v", err)
	}
	oid, ok := oidRes.([]byte)
	if !ok || len(oid) == 0 {
		t.Fatalf("ObjectWrite: expected oid []byte, got %T %v", oidRes, oidRes)
	}

	row, err := q.ObjectRead(ctx, ObjectReadParams{RepoID: repoID, Oid: oid})
	if err != nil {
		t.Fatalf("ObjectRead: %v", err)
	}
	if row.Type != 3 || row.Size != int32(len(content)) || string(row.Content) != string(content) {
		t.Errorf("ObjectRead: type=%d size=%d content=%q", row.Type, row.Size, row.Content)
	}
}

func TestObjectWriteIdempotent(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_idem_repo")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	content := []byte("duplicate test")
	oid1, err := q.ObjectWrite(ctx, ObjectWriteParams{RepoID: repoID, ObjType: 3, Content: content})
	if err != nil {
		t.Fatalf("ObjectWrite 1: %v", err)
	}
	oid2, err := q.ObjectWrite(ctx, ObjectWriteParams{RepoID: repoID, ObjType: 3, Content: content})
	if err != nil {
		t.Fatalf("ObjectWrite 2: %v", err)
	}
	o1, o2 := oid1.([]byte), oid2.([]byte)
	if string(o1) != string(o2) {
		t.Errorf("idempotent: oid1 %x != oid2 %x", o1, o2)
	}
}

func TestObjectReadNonexistent(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_nonex_repo")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	zeroOid := make([]byte, 20)
	_, err = q.ObjectRead(ctx, ObjectReadParams{RepoID: repoID, Oid: zeroOid})
	if err == nil {
		t.Error("ObjectRead nonexistent: expected error")
	}
}

func TestObjectWriteMultipleTypes(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_multi_repo")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	blobOid, err := q.ObjectWrite(ctx, ObjectWriteParams{RepoID: repoID, ObjType: 3, Content: []byte("blob content")})
	if err != nil {
		t.Fatalf("blob write: %v", err)
	}
	commitContent := []byte("tree 0000000000000000000000000000000000000000\nauthor Test <test@test.com> 1234567890 +0000\ncommitter Test <test@test.com> 1234567890 +0000\n\ntest commit")
	commitOid, err := q.ObjectWrite(ctx, ObjectWriteParams{RepoID: repoID, ObjType: 1, Content: commitContent})
	if err != nil {
		t.Fatalf("commit write: %v", err)
	}
	if string(blobOid.([]byte)) == string(commitOid.([]byte)) {
		t.Error("blob and commit OIDs should differ")
	}

	row, _ := q.ObjectRead(ctx, ObjectReadParams{RepoID: repoID, Oid: blobOid.([]byte)})
	if row.Type != 3 {
		t.Errorf("blob type: got %d", row.Type)
	}
	row, _ = q.ObjectRead(ctx, ObjectReadParams{RepoID: repoID, Oid: commitOid.([]byte)})
	if row.Type != 1 {
		t.Errorf("commit type: got %d", row.Type)
	}
}
