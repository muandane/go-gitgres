// Package backend provides shared logic for the gitgres CLI and remote-helper protocol.
package backend

import (
	"context"
	"fmt"
	"strings"

	"go-gitgres/internal/storer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// ParseURL splits "conninfo/reponame" into conninfo and reponame.
func ParseURL(url string) (conninfo, reponame string, err error) {
	i := strings.LastIndex(url, "/")
	if i <= 0 || i == len(url)-1 {
		return "", "", fmt.Errorf("invalid URL: expected <conninfo>/<reponame>, got %q", url)
	}
	return url[:i], url[i+1:], nil
}

// RefLine is one ref for listing (hash or symbolic).
type RefLine struct {
	Hash     string
	Name     string
	Symbolic string
}

// ListRefs returns all refs from the storer for the caller to format/print.
// Uses a single DB call and pre-allocates the result slice.
func ListRefs(ctx context.Context, s *storer.PostgresStorer) ([]RefLine, error) {
	rows, err := s.ListRefsRows()
	if err != nil {
		return nil, err
	}
	out := make([]RefLine, 0, len(rows))
	for _, r := range rows {
		if len(r.Oid) == 20 {
			var h plumbing.Hash
			copy(h[:], r.Oid)
			out = append(out, RefLine{Hash: h.String(), Name: r.Name})
		} else if r.SymbolicValid {
			out = append(out, RefLine{Symbolic: r.Symbolic, Name: r.Name})
		}
	}
	return out, nil
}

// CopyObjectsFromRepoToStorer copies all objects from a local repo into the Postgres storer.
// Returns the number of objects copied.
func CopyObjectsFromRepoToStorer(repo *git.Repository, pgStorer *storer.PostgresStorer) (int, error) {
	iter, err := repo.Storer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	n := 0
	err = iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, err := pgStorer.SetEncodedObject(obj)
		if err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

// CopyRefsFromRepoToStorer copies all refs from a local repo into the Postgres storer.
// Returns the number of refs copied.
func CopyRefsFromRepoToStorer(repo *git.Repository, pgStorer *storer.PostgresStorer) (int, error) {
	iter, err := repo.Storer.IterReferences()
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	n := 0
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if err := pgStorer.SetReference(ref); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

// CopyFromStorerToRepo copies all objects and refs from the Postgres storer into the repo.
// Returns (objectCount, refCount, error).
func CopyFromStorerToRepo(ctx context.Context, pgStorer *storer.PostgresStorer, repo *git.Repository) (objN, refN int, err error) {
	iter, err := pgStorer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		return 0, 0, err
	}
	defer iter.Close()
	err = iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, e := repo.Storer.SetEncodedObject(obj)
		if e != nil {
			return e
		}
		objN++
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	refIter, err := pgStorer.IterReferences()
	if err != nil {
		return 0, 0, err
	}
	defer refIter.Close()
	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if e := repo.Storer.SetReference(ref); e != nil {
			return e
		}
		refN++
		return nil
	})
	return objN, refN, err
}
