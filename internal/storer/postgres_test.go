package storer

import (
	"context"
	"io"
	"testing"

	"go-gitgres/internal/db"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
)

func TestPostgresStorerRoundtrip(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	s, err := NewPostgresStorer(ctx, pool, "go_storer_roundtrip")
	if err != nil {
		t.Fatalf("NewPostgresStorer: %v", err)
	}

	content := []byte("hello storer")
	obj := s.NewEncodedObject().(*plumbing.MemoryObject)
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(content)))
	obj.Write(content)

	hash, err := s.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("SetEncodedObject: %v", err)
	}
	if hash.IsZero() {
		t.Fatal("hash is zero")
	}

	got, err := s.EncodedObject(plumbing.BlobObject, hash)
	if err != nil {
		t.Fatalf("EncodedObject: %v", err)
	}
	r, _ := got.Reader()
	defer r.Close()
	b, _ := io.ReadAll(r)
	if string(b) != string(content) {
		t.Errorf("content: got %q", b)
	}

	ref := plumbing.NewHashReference("refs/heads/main", hash)
	if err := s.SetReference(ref); err != nil {
		t.Fatalf("SetReference: %v", err)
	}
	gotRef, err := s.Reference("refs/heads/main")
	if err != nil {
		t.Fatalf("Reference: %v", err)
	}
	if gotRef.Hash() != hash {
		t.Errorf("ref hash: got %s", gotRef.Hash())
	}

	if id := s.RepoID(); id <= 0 {
		t.Errorf("RepoID = %d, want positive", id)
	}
}

func TestPostgresStorer_NewPostgresStorerWithID_RepoID(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	s1, err := NewPostgresStorer(ctx, pool, "go_storer_withid_test")
	if err != nil {
		t.Fatalf("NewPostgresStorer: %v", err)
	}
	repoID := s1.RepoID()
	if repoID <= 0 {
		t.Fatalf("RepoID = %d, want positive", repoID)
	}

	s2 := NewPostgresStorerWithID(ctx, pool, repoID)
	if s2.RepoID() != repoID {
		t.Errorf("NewPostgresStorerWithID RepoID = %d, want %d", s2.RepoID(), repoID)
	}
	refs, err := s2.IterReferences()
	if err != nil {
		t.Fatalf("IterReferences (storer from WithID): %v", err)
	}
	refs.Close()
}

func TestPostgresStorer_IterEncodedObjects_HasEncodedObject_EncodedObjectSize(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	s, err := NewPostgresStorer(ctx, pool, "go_storer_iter_test")
	if err != nil {
		t.Fatalf("NewPostgresStorer: %v", err)
	}

	content := []byte("iter test blob")
	obj := s.NewEncodedObject().(*plumbing.MemoryObject)
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(content)))
	obj.Write(content)
	hash, err := s.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("SetEncodedObject: %v", err)
	}

	iter, err := s.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		t.Fatalf("IterEncodedObjects: %v", err)
	}
	var count int
	err = iter.ForEach(func(plumbing.EncodedObject) error { count++; return nil })
	iter.Close()
	if err != nil {
		t.Fatalf("IterEncodedObjects ForEach: %v", err)
	}
	if count != 1 {
		t.Errorf("IterEncodedObjects: count = %d, want 1", count)
	}

	if err := s.HasEncodedObject(hash); err != nil {
		t.Errorf("HasEncodedObject: %v", err)
	}
	if err := s.HasEncodedObject(plumbing.ZeroHash); err != plumbing.ErrObjectNotFound {
		t.Errorf("HasEncodedObject(zero): want ErrObjectNotFound, got %v", err)
	}

	size, err := s.EncodedObjectSize(hash)
	if err != nil {
		t.Fatalf("EncodedObjectSize: %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("EncodedObjectSize: got %d, want %d", size, len(content))
	}
}

func TestPostgresStorer_IterReferences_RemoveReference_CheckAndSetReference_Symbolic(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	s, err := NewPostgresStorer(ctx, pool, "go_storer_refs_test")
	if err != nil {
		t.Fatalf("NewPostgresStorer: %v", err)
	}

	content := []byte("ref test")
	obj := s.NewEncodedObject().(*plumbing.MemoryObject)
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(content)))
	obj.Write(content)
	hash, err := s.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("SetEncodedObject: %v", err)
	}

	mainRef := plumbing.NewHashReference("refs/heads/main", hash)
	if err := s.SetReference(mainRef); err != nil {
		t.Fatalf("SetReference(main): %v", err)
	}
	symRef := plumbing.NewSymbolicReference("HEAD", "refs/heads/main")
	if err := s.SetReference(symRef); err != nil {
		t.Fatalf("SetReference(HEAD symbolic): %v", err)
	}

	refIter, err := s.IterReferences()
	if err != nil {
		t.Fatalf("IterReferences: %v", err)
	}
	var refNames []string
	err = refIter.ForEach(func(r *plumbing.Reference) error {
		refNames = append(refNames, string(r.Name()))
		return nil
	})
	refIter.Close()
	if err != nil {
		t.Fatalf("IterReferences ForEach: %v", err)
	}
	if len(refNames) != 2 {
		t.Errorf("IterReferences: got %v, want 2 refs", refNames)
	}

	if err := s.CheckAndSetReference(mainRef, nil); err != nil {
		t.Errorf("CheckAndSetReference(main, nil): %v", err)
	}
	// Use a non-zero wrong hash: git_ref_update treats zero old_oid as "no CAS" and succeeds.
	var wrongHash plumbing.Hash
	for i := range wrongHash {
		wrongHash[i] = 0x11
	}
	wrongOld := plumbing.NewHashReference("refs/heads/main", wrongHash)
	if err := s.CheckAndSetReference(mainRef, wrongOld); err != storage.ErrReferenceHasChanged {
		t.Errorf("CheckAndSetReference with wrong old: want ErrReferenceHasChanged, got %v", err)
	}

	if err := s.RemoveReference("refs/heads/main"); err != nil {
		t.Fatalf("RemoveReference: %v", err)
	}
	_, err = s.Reference("refs/heads/main")
	if err != plumbing.ErrReferenceNotFound {
		t.Errorf("Reference after Remove: want ErrReferenceNotFound, got %v", err)
	}
}
