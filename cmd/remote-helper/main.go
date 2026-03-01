// Command git-remote-gitgres: Git remote helper that stores objects and refs in PostgreSQL.
//
// Git invokes: git-remote-gitgres <remote-name> <url>
// URL format: <conninfo>/<reponame>  e.g. dbname=gitgres_test/myrepo
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"go-gitgres/internal/db"
	"go-gitgres/internal/storer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitstorer "github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fatal: git-remote-gitgres: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: git-remote-gitgres <remote-name> <url>\n")
		fmt.Fprintf(os.Stderr, "  git remote add <name> gitgres::<conninfo>/<reponame>\n")
		fmt.Fprintf(os.Stderr, "  git push <name> main\n")
		fmt.Fprintf(os.Stderr, "  git clone gitgres::<conninfo>/<reponame>\n")
		os.Exit(1)
	}
	url := os.Args[2]
	conninfo, reponame, err := parseURL(url)
	if err != nil {
		die("%v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, conninfo)
	if err != nil {
		die("connection failed: %v", err)
	}
	defer pool.Close()

	pgStorer, err := storer.NewPostgresStorer(ctx, pool, reponame)
	if err != nil {
		die("open storer: %v", err)
	}

	gitDir := os.Getenv("GIT_DIR")
	if gitDir == "" {
		gitDir = ".git"
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\n")
		if line == "" {
			break
		}
		switch {
		case line == "capabilities":
			fmt.Print("fetch\npush\n\n")
			os.Stdout.Sync()
		case line == "list" || line == "list for-push":
			cmdList(pgStorer)
		case strings.HasPrefix(line, "fetch "):
			readFetchLines(scanner)
			cmdFetch(ctx, pgStorer, gitDir)
		case strings.HasPrefix(line, "push "):
			specs := parsePushLine(line)
			readPushLines(scanner, &specs)
			cmdPush(ctx, pool, pgStorer, gitDir, specs)
		}
	}
	if err := scanner.Err(); err != nil {
		die("read stdin: %v", err)
	}
}

func parseURL(url string) (conninfo, reponame string, err error) {
	i := strings.LastIndex(url, "/")
	if i <= 0 || i == len(url)-1 {
		return "", "", fmt.Errorf("invalid URL: expected <conninfo>/<reponame>, got %q", url)
	}
	return url[:i], url[i+1:], nil
}

func cmdList(s *storer.PostgresStorer) {
	refIter, err := s.IterReferences()
	if err != nil {
		die("list refs: %v", err)
	}
	defer refIter.Close()

	var headOid, headSymbolic string
	refIter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name() == "HEAD" {
			if ref.Type() == plumbing.SymbolicReference {
				headSymbolic = string(ref.Target())
			} else {
				headOid = ref.Hash().String()
			}
			return nil
		}
		if ref.Type() == plumbing.HashReference {
			fmt.Printf("%s %s\n", ref.Hash(), ref.Name())
		}
		return nil
	})
	if headSymbolic != "" {
		fmt.Printf("@%s HEAD\n", headSymbolic)
	} else if headOid != "" {
		fmt.Printf("%s HEAD\n", headOid)
	}
	fmt.Println()
	os.Stdout.Sync()
}

func readFetchLines(scanner *bufio.Scanner) {
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			break
		}
	}
}

func cmdFetch(ctx context.Context, pgStorer *storer.PostgresStorer, gitDir string) {
	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		die("open local repo: %v", err)
	}

	iter, err := pgStorer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		die("iterate objects: %v", err)
	}
	count := 0
	iter.ForEach(func(obj plumbing.EncodedObject) error {
		if err := repo.Storer.HasEncodedObject(obj.Hash()); err == nil {
			return nil
		}
		_, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			return err
		}
		count++
		return nil
	})
	iter.Close()
	fmt.Println()
	os.Stdout.Sync()
	_ = count
}

type pushSpec struct {
	src string
	dst string
}

func parsePushLine(line string) (specs []pushSpec) {
	if !strings.HasPrefix(line, "push ") {
		return nil
	}
	rest := strings.TrimSpace(line[5:])
	if strings.HasPrefix(rest, "+") {
		rest = rest[1:]
	}
	before, after, ok := strings.Cut(rest, ":")
	if !ok {
		return append(specs, pushSpec{dst: rest})
	}
	return append(specs, pushSpec{src: before, dst: after})
}

func readPushLines(scanner *bufio.Scanner, specs *[]pushSpec) {
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			break
		}
		if strings.HasPrefix(line, "push ") {
			*specs = append(*specs, parsePushLine(line)...)
		}
	}
}

func cmdPush(ctx context.Context, pool *pgxpool.Pool, pgStorer *storer.PostgresStorer, gitDir string, specs []pushSpec) {
	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		die("open local repo: %v", err)
	}

	// Copy all local objects to PG
	iter, err := repo.Storer.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		die("iterate objects: %v", err)
	}
	iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, err := pgStorer.SetEncodedObject(obj)
		return err
	})
	iter.Close()

	q := db.New(pool)
	repoID := pgStorer.RepoID()

	for _, spec := range specs {
		if spec.src == "" {
			if err := q.RefDelete(ctx, db.RefDeleteParams{RepoID: repoID, Name: spec.dst}); err != nil {
				fmt.Printf("error %s %v\n", spec.dst, err)
				continue
			}
			fmt.Printf("ok %s\n", spec.dst)
			continue
		}

		ref, err := repo.Storer.Reference(plumbing.ReferenceName(spec.src))
		if err != nil {
			ref, err = gitstorer.ResolveReference(repo.Storer, plumbing.ReferenceName(spec.src))
		}
		if err != nil {
			if plumbing.IsHash(spec.src) {
				h := plumbing.NewHash(spec.src)
				ref = plumbing.NewHashReference(plumbing.ReferenceName(spec.dst), h)
				err = nil
			}
		}
		if err != nil && ref == nil {
			fmt.Printf("error %s cannot resolve '%s'\n", spec.dst, spec.src)
			continue
		}
		if ref != nil && ref.Type() == plumbing.HashReference {
			h := ref.Hash()
			err = q.RefInsert(ctx, db.RefInsertParams{RepoID: repoID, Name: spec.dst, Oid: h[:]})
			if err != nil {
				fmt.Printf("error %s %v\n", spec.dst, err)
			} else {
				fmt.Printf("ok %s\n", spec.dst)
			}
		}
	}

	// Ensure HEAD exists
	_, err = q.RefsHasHEAD(ctx, repoID)
	if err != nil && len(specs) > 0 {
		_ = q.RefInsertSymbolic(ctx, db.RefInsertSymbolicParams{
			RepoID: repoID, Name: "HEAD", Symbolic: pgtype.Text{String: specs[0].dst, Valid: true},
		})
	}

	fmt.Println()
	os.Stdout.Sync()
}
