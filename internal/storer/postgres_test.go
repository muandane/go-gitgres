package storer

import (
	"context"
	"io"
	"testing"

	"go-gitgres/internal/db"

	"github.com/go-git/go-git/v5/plumbing"
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
}
