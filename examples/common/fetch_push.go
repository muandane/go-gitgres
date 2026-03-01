package common

import (
	"context"
	"os"

	"go-gitgres/internal/backend"
	"go-gitgres/internal/storer"

	"github.com/go-git/go-git/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FetchAndPushToGitgres clones gitURL into a temp dir, pushes that repo to Postgres
// under the given conninfo and reponame, then returns a cleanup function.
// The repo is created in Postgres if it does not exist.
func FetchAndPushToGitgres(ctx context.Context, conninfo, reponame, gitURL string) (cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "gitgres-fetch-*")
	if err != nil {
		return nil, err
	}
	done := false
	defer func() {
		if !done {
			_ = os.RemoveAll(dir)
		}
	}()

	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: gitURL,
	})
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, conninfo)
	if err != nil {
		return nil, err
	}

	pgStorer, err := storer.NewPostgresStorer(ctx, pool, reponame)
	if err != nil {
		pool.Close()
		return nil, err
	}

	repo, err := git.PlainOpen(dir)
	if err != nil {
		pool.Close()
		return nil, err
	}

	if _, err := backend.CopyObjectsFromRepoToStorer(repo, pgStorer); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := backend.CopyRefsFromRepoToStorer(repo, pgStorer); err != nil {
		pool.Close()
		return nil, err
	}

	done = true
	return func() {
		pool.Close()
		_ = os.RemoveAll(dir)
	}, nil
}
