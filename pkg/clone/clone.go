package clone

import (
	"context"
	"os"

	"github.com/muandane/go-gitgres/internal/db"

	"github.com/muandane/go-gitgres/internal/storer"

	"github.com/muandane/go-gitgres/internal/backend"

	"github.com/go-git/go-git/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CloneFromGitgres clones the repository named reponame from Postgres (conninfo)
// into a temp dir and returns that dir and a cleanup function.
// The worktree is checked out so the directory contains files for scanning.
func CloneFromGitgres(ctx context.Context, conninfo, reponame string) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "gitgres-clone-*")
	if err != nil {
		return "", nil, err
	}
	done := false
	defer func() {
		if !done {
			_ = os.RemoveAll(dir)
		}
	}()

	pool, err := pgxpool.New(ctx, conninfo)
	if err != nil {
		return "", nil, err
	}

	q := db.New(pool)
	repoID, err := q.GetRepo(ctx, reponame)
	if err != nil {
		pool.Close()
		return "", nil, err
	}

	pgStorer := storer.NewPostgresStorerWithID(ctx, pool, repoID)
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		pool.Close()
		return "", nil, err
	}

	if _, _, err = backend.CopyFromStorerToRepo(ctx, pgStorer, repo); err != nil {
		pool.Close()
		return "", nil, err
	}

	head, err := repo.Head()
	if err == nil {
		w, wErr := repo.Worktree()
		if wErr == nil {
			_ = w.Checkout(&git.CheckoutOptions{Branch: head.Name()})
		}
	}

	done = true
	return dir, func() {
		pool.Close()
		_ = os.RemoveAll(dir)
	}, nil
}
