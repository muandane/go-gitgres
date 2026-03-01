package db

import (
	"context"
	"testing"
)

func TestRefCreate(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_ref_create")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	oid := make([]byte, 20)
	for i := range oid {
		oid[i] = 0xaa
	}
	okRes, err := q.RefUpdate(ctx, RefUpdateParams{
		RepoID: repoID,
		Name:   "refs/heads/main",
		NewOid: oid,
		OldOid: nil,
		Force:  false,
	})
	if err != nil {
		t.Fatalf("RefUpdate create: %v", err)
	}
	if ok, _ := okRes.(bool); !ok {
		t.Error("RefUpdate create: expected true")
	}

	refs, err := q.ListRefs(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefs: %v", err)
	}
	var found bool
	for _, r := range refs {
		if r.Name == "refs/heads/main" && len(r.Oid) == 20 {
			found = true
			break
		}
	}
	if !found {
		t.Error("ref not found after create")
	}
}

func TestRefUpdateWithCAS(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_ref_cas")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	oldOid := make([]byte, 20)
	newOid := make([]byte, 20)
	for i := range oldOid {
		oldOid[i] = 0xaa
	}
	for i := range newOid {
		newOid[i] = 0xbb
	}

	q.RefUpdate(ctx, RefUpdateParams{RepoID: repoID, Name: "refs/heads/main", NewOid: oldOid})
	okRes, err := q.RefUpdate(ctx, RefUpdateParams{
		RepoID: repoID,
		Name:   "refs/heads/main",
		NewOid: newOid,
		OldOid: oldOid,
		Force:  false,
	})
	if err != nil {
		t.Fatalf("RefUpdate CAS: %v", err)
	}
	if ok, _ := okRes.(bool); !ok {
		t.Error("RefUpdate CAS: expected true")
	}
}

func TestRefUpdateCASFails(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_ref_cas_fail")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	oldOid := make([]byte, 20)
	newOid := make([]byte, 20)
	wrongOid := make([]byte, 20)
	for i := range oldOid {
		oldOid[i] = 0xaa
	}
	for i := range newOid {
		newOid[i] = 0xbb
	}
	for i := range wrongOid {
		wrongOid[i] = 0xcc
	}

	q.RefUpdate(ctx, RefUpdateParams{RepoID: repoID, Name: "refs/heads/main", NewOid: oldOid})
	okRes, err := q.RefUpdate(ctx, RefUpdateParams{
		RepoID: repoID,
		Name:   "refs/heads/main",
		NewOid: newOid,
		OldOid: wrongOid,
		Force:  false,
	})
	if err != nil {
		t.Fatalf("RefUpdate: %v", err)
	}
	if ok, _ := okRes.(bool); ok {
		t.Error("RefUpdate CAS wrong old: expected false")
	}
}

func TestRefForceUpdate(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_ref_force")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	oldOid := make([]byte, 20)
	newOid := make([]byte, 20)
	wrongOid := make([]byte, 20)
	for i := range oldOid {
		oldOid[i] = 0xaa
	}
	for i := range newOid {
		newOid[i] = 0xbb
	}
	for i := range wrongOid {
		wrongOid[i] = 0xcc
	}

	q.RefUpdate(ctx, RefUpdateParams{RepoID: repoID, Name: "refs/heads/main", NewOid: oldOid})
	okRes, err := q.RefUpdate(ctx, RefUpdateParams{
		RepoID: repoID,
		Name:   "refs/heads/main",
		NewOid: newOid,
		OldOid: wrongOid,
		Force:  true,
	})
	if err != nil {
		t.Fatalf("RefUpdate force: %v", err)
	}
	if ok, _ := okRes.(bool); !ok {
		t.Error("RefUpdate force: expected true")
	}
}

func TestRefSetSymbolic(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_ref_sym")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	err = q.RefSetSymbolic(ctx, RefSetSymbolicParams{
		RepoID: repoID,
		Name:   "HEAD",
		Target: "refs/heads/main",
	})
	if err != nil {
		t.Fatalf("RefSetSymbolic: %v", err)
	}

	refs, err := q.ListRefs(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefs: %v", err)
	}
	for _, r := range refs {
		if r.Name == "HEAD" {
			if !r.Symbolic.Valid || r.Symbolic.String != "refs/heads/main" {
				t.Errorf("HEAD symbolic: got %v", r.Symbolic)
			}
			return
		}
	}
	t.Error("HEAD ref not found")
}

func TestRefDelete(t *testing.T) {
	ctx := context.Background()
	pool := OpenPoolForTest(ctx, t, "")
	defer pool.Close()

	q := New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, "go_test_ref_del")
	if err != nil {
		t.Fatalf("GetOrCreateRepo: %v", err)
	}

	oid := make([]byte, 20)
	for i := range oid {
		oid[i] = 0xaa
	}
	q.RefUpdate(ctx, RefUpdateParams{RepoID: repoID, Name: "refs/heads/main", NewOid: oid})

	zeroOid := make([]byte, 20)
	okRes, err := q.RefUpdate(ctx, RefUpdateParams{
		RepoID: repoID,
		Name:   "refs/heads/main",
		NewOid: zeroOid,
		OldOid: nil,
		Force:  true,
	})
	if err != nil {
		t.Fatalf("RefUpdate delete: %v", err)
	}
	if ok, _ := okRes.(bool); !ok {
		t.Error("RefUpdate delete: expected true")
	}

	refs, err := q.ListRefs(ctx, repoID)
	if err != nil {
		t.Fatalf("ListRefs: %v", err)
	}
	for _, r := range refs {
		if r.Name == "refs/heads/main" {
			t.Error("ref should be deleted")
		}
	}
}
