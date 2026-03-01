// Command gitgres-backend: CLI for moving git objects between local repos and PostgreSQL.
//
//	gitgres-backend init     <conninfo> <reponame>
//	gitgres-backend push     <conninfo> <reponame> <local-repo-path>
//	gitgres-backend clone    <conninfo> <reponame> <dest-dir>
//	gitgres-backend ls-refs  <conninfo> <reponame>
package main

import (
	"context"
	"fmt"
	"os"

	"go-gitgres/internal/db"
	"go-gitgres/internal/storer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jackc/pgx/v5/pgxpool"
)

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fatal: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		die("usage: gitgres-backend init|push|clone|ls-refs [args...]")
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	ctx := context.Background()

	switch cmd {
	case "init":
		if len(args) < 2 {
			die("usage: gitgres-backend init <conninfo> <reponame>")
		}
		runInit(ctx, args[0], args[1])
	case "push":
		if len(args) < 3 {
			die("usage: gitgres-backend push <conninfo> <reponame> <local-repo-path>")
		}
		runPush(ctx, args[0], args[1], args[2])
	case "clone":
		if len(args) < 3 {
			die("usage: gitgres-backend clone <conninfo> <reponame> <dest-dir>")
		}
		runClone(ctx, args[0], args[1], args[2])
	case "ls-refs":
		if len(args) < 2 {
			die("usage: gitgres-backend ls-refs <conninfo> <reponame>")
		}
		runLsRefs(ctx, args[0], args[1])
	default:
		die("unknown command: %s", cmd)
	}
}

func runInit(ctx context.Context, conninfo, reponame string) {
	pool, err := pgxpool.New(ctx, conninfo)
	if err != nil {
		die("connection failed: %v", err)
	}
	defer pool.Close()
	q := db.New(pool)
	repoID, err := q.GetOrCreateRepo(ctx, reponame)
	if err != nil {
		die("get_or_create_repo: %v", err)
	}
	fmt.Printf("Repository '%s' ready (id=%d)\n", reponame, repoID)
}

func runPush(ctx context.Context, conninfo, reponame, localPath string) {
	pool, err := pgxpool.New(ctx, conninfo)
	if err != nil {
		die("connection failed: %v", err)
	}
	defer pool.Close()

	pgStorer, err := storer.NewPostgresStorer(ctx, pool, reponame)
	if err != nil {
		die("open pg storer: %v", err)
	}

	repo, err := git.PlainOpen(localPath)
	if err != nil {
		die("open local repo: %v", err)
	}

	objCount := 0
	iter, err := repo.Storer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		die("iterate objects: %v", err)
	}
	err = iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, err := pgStorer.SetEncodedObject(obj)
		if err != nil {
			return err
		}
		objCount++
		return nil
	})
	iter.Close()
	if err != nil {
		die("copy objects: %v", err)
	}

	refIter, err := repo.Storer.IterReferences()
	if err != nil {
		die("iterate refs: %v", err)
	}
	refCount := 0
	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if err := pgStorer.SetReference(ref); err != nil {
			return err
		}
		refCount++
		return nil
	})
	refIter.Close()
	if err != nil {
		die("copy refs: %v", err)
	}

	fmt.Printf("Pushed %d objects\n", objCount)
	fmt.Printf("Pushed %d refs\n", refCount)
}

func runClone(ctx context.Context, conninfo, reponame, destPath string) {
	pool, err := pgxpool.New(ctx, conninfo)
	if err != nil {
		die("connection failed: %v", err)
	}
	defer pool.Close()

	q := db.New(pool)
	repoID, err := q.GetRepo(ctx, reponame)
	if err != nil {
		die("repository '%s' not found", reponame)
	}

	pgStorer := storer.NewPostgresStorerWithID(ctx, pool, repoID)

	repo, err := git.PlainInit(destPath, false)
	if err != nil {
		die("init local repo: %v", err)
	}

	iter, err := pgStorer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		die("iterate objects: %v", err)
	}
	objCount := 0
	err = iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			return err
		}
		objCount++
		return nil
	})
	iter.Close()
	if err != nil {
		die("copy objects: %v", err)
	}

	refIter, err := pgStorer.IterReferences()
	if err != nil {
		die("iterate refs: %v", err)
	}
	refCount := 0
	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if err := repo.Storer.SetReference(ref); err != nil {
			return err
		}
		refCount++
		return nil
	})
	refIter.Close()
	if err != nil {
		die("copy refs: %v", err)
	}

	fmt.Printf("Cloned %d objects\n", objCount)
	fmt.Printf("Cloned %d refs\n", refCount)
}

func runLsRefs(ctx context.Context, conninfo, reponame string) {
	pool, err := pgxpool.New(ctx, conninfo)
	if err != nil {
		die("connection failed: %v", err)
	}
	defer pool.Close()

	pgStorer, err := storer.NewPostgresStorer(ctx, pool, reponame)
	if err != nil {
		die("open storer: %v", err)
	}

	refIter, err := pgStorer.IterReferences()
	if err != nil {
		die("list refs: %v", err)
	}
	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.HashReference {
			fmt.Printf("%s %s\n", ref.Hash(), ref.Name())
		} else {
			fmt.Printf("@%s %s\n", ref.Target(), ref.Name())
		}
		return nil
	})
	refIter.Close()
	if err != nil {
		die("list refs: %v", err)
	}
}
