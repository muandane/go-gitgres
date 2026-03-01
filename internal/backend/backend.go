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
func ListRefs(ctx context.Context, s *storer.PostgresStorer) ([]RefLine, error) {
	iter, err := s.IterReferences()
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var out []RefLine
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			out = append(out, RefLine{Hash: ref.Hash().String(), Name: string(ref.Name())})
		} else {
			out = append(out, RefLine{Symbolic: string(ref.Target()), Name: string(ref.Name())})
		}
		return nil
	})
	return out, err
}

// CopyObjectsFromRepoToStorer copies all objects from a local repo into the Postgres storer.
func CopyObjectsFromRepoToStorer(repo *git.Repository, pgStorer *storer.PostgresStorer) error {
	iter, err := repo.Storer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		return err
	}
	defer iter.Close()
	return iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, err := pgStorer.SetEncodedObject(obj)
		return err
	})
}

// CopyRefsFromRepoToStorer copies all refs from a local repo into the Postgres storer.
func CopyRefsFromRepoToStorer(repo *git.Repository, pgStorer *storer.PostgresStorer) error {
	iter, err := repo.Storer.IterReferences()
	if err != nil {
		return err
	}
	defer iter.Close()
	return iter.ForEach(func(ref *plumbing.Reference) error {
		return pgStorer.SetReference(ref)
	})
}

// CopyFromStorerToRepo copies all objects and refs from the Postgres storer into the repo.
func CopyFromStorerToRepo(ctx context.Context, pgStorer *storer.PostgresStorer, repo *git.Repository) error {
	iter, err := pgStorer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		return err
	}
	defer iter.Close()
	if err := iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, err := repo.Storer.SetEncodedObject(obj)
		return err
	}); err != nil {
		return err
	}
	refIter, err := pgStorer.IterReferences()
	if err != nil {
		return err
	}
	defer refIter.Close()
	return refIter.ForEach(func(ref *plumbing.Reference) error {
		return repo.Storer.SetReference(ref)
	})
}
