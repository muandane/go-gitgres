package backend

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go-gitgres/internal/db"
	"go-gitgres/internal/storer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		url      string
		wantConn string
		wantRepo string
		wantErr  bool
	}{
		{"dbname=gitgres_test/myrepo", "dbname=gitgres_test", "myrepo", false},
		{"host=localhost dbname=foo/bar", "host=localhost dbname=foo", "bar", false},
		{"a/b/c", "a/b", "c", false},
		{"nopath", "", "", true},
		{"/onlyslash", "", "", true},
		{"trailing/", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		conn, repo, err := ParseURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseURL(%q) err = %v, wantErr %v", tt.url, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (conn != tt.wantConn || repo != tt.wantRepo) {
			t.Errorf("ParseURL(%q) = %q, %q; want %q, %q", tt.url, conn, repo, tt.wantConn, tt.wantRepo)
		}
	}
}

func TestListRefsAndCopyRoundtrip(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	pgStorer, err := storer.NewPostgresStorer(ctx, pool, "go_backend_listrefs_copy_test")
	if err != nil {
		t.Fatalf("NewPostgresStorer: %v", err)
	}

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("backend test"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := w.Add("f.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := w.Commit("first", &git.CommitOptions{Author: &object.Signature{Name: "t", Email: "t@t"}}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	objCount, err := CopyObjectsFromRepoToStorer(repo, pgStorer)
	if err != nil {
		t.Fatalf("CopyObjectsFromRepoToStorer: %v", err)
	}
	if objCount == 0 {
		t.Error("CopyObjectsFromRepoToStorer: expected at least one object")
	}
	refCount, err := CopyRefsFromRepoToStorer(repo, pgStorer)
	if err != nil {
		t.Fatalf("CopyRefsFromRepoToStorer: %v", err)
	}
	if refCount == 0 {
		t.Error("CopyRefsFromRepoToStorer: expected at least one ref")
	}

	refs, err := ListRefs(ctx, pgStorer)
	if err != nil {
		t.Fatalf("ListRefs: %v", err)
	}
	if len(refs) == 0 {
		t.Error("ListRefs: expected at least one ref")
	}

	destDir := t.TempDir()
	destRepo, err := git.PlainInit(destDir, false)
	if err != nil {
		t.Fatalf("PlainInit dest: %v", err)
	}
	gotObjN, gotRefN, err := CopyFromStorerToRepo(ctx, pgStorer, destRepo)
	if err != nil {
		t.Fatalf("CopyFromStorerToRepo: %v", err)
	}
	if gotObjN != objCount {
		t.Errorf("CopyFromStorerToRepo objects: got %d, want %d", gotObjN, objCount)
	}
	if gotRefN != refCount {
		t.Errorf("CopyFromStorerToRepo refs: got %d, want %d", gotRefN, refCount)
	}

	destRefs, err := destRepo.Storer.IterReferences()
	if err != nil {
		t.Fatalf("dest IterReferences: %v", err)
	}
	var n int
	_ = destRefs.ForEach(func(*plumbing.Reference) error { n++; return nil })
	destRefs.Close()
	if n == 0 {
		t.Error("dest repo has no refs after CopyFromStorerToRepo")
	}
}
