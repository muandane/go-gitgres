// Command gitgres-backend: CLI and Git remote helper for storing Git in PostgreSQL.
//
// CLI:
//
//	gitgres-backend init     <conninfo> <reponame>
//	gitgres-backend push     <conninfo> <reponame> <local-repo-path>
//	gitgres-backend clone    <conninfo> <reponame> <dest-dir>
//	gitgres-backend ls-refs  <conninfo> <reponame>
//
// Remote helper (when invoked as git-remote-gitgres <name> <url>):
//
//	git remote add pg gitgres::<conninfo>/<reponame>
//	git push pg main   /   git clone gitgres::<conninfo>/<reponame>
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"go-gitgres/internal/backend"
	"go-gitgres/internal/db"
	"go-gitgres/internal/storer"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitstorer "github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// exitFunc is called by die; tests override it to panic instead of os.Exit.
var exitFunc = os.Exit

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fatal: "+format+"\n", args...)
	exitFunc(1)
}

func main() {
	// Remote-helper mode: git-remote-gitgres <remote-name> <url>
	if len(os.Args) == 3 && strings.Contains(os.Args[2], "/") {
		runRemoteHelper()
		return
	}
	runCLI()
}

func runCLI() {
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
	objCount, err := backend.CopyObjectsFromRepoToStorer(repo, pgStorer)
	if err != nil {
		die("copy objects: %v", err)
	}
	refCount, err := backend.CopyRefsFromRepoToStorer(repo, pgStorer)
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
	objN, refN, err := backend.CopyFromStorerToRepo(ctx, pgStorer, repo)
	if err != nil {
		die("copy from storer: %v", err)
	}
	fmt.Printf("Cloned %d objects\n", objN)
	fmt.Printf("Cloned %d refs\n", refN)
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
	refs, err := backend.ListRefs(ctx, pgStorer)
	if err != nil {
		die("list refs: %v", err)
	}
	for _, r := range refs {
		if r.Symbolic != "" {
			fmt.Printf("@%s %s\n", r.Symbolic, r.Name)
		} else {
			fmt.Printf("%s %s\n", r.Hash, r.Name)
		}
	}
}

// --- Remote helper protocol ---

func runRemoteHelper() {
	url := os.Args[2]
	conninfo, reponame, err := backend.ParseURL(url)
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
			cmdList(ctx, pgStorer)
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

func cmdList(ctx context.Context, s *storer.PostgresStorer) {
	refs, err := backend.ListRefs(ctx, s)
	if err != nil {
		die("list refs: %v", err)
	}
	var headOid, headSymbolic string
	for _, r := range refs {
		if r.Name == "HEAD" {
			if r.Symbolic != "" {
				headSymbolic = r.Symbolic
			} else {
				headOid = r.Hash
			}
			continue
		}
		if r.Symbolic == "" {
			fmt.Printf("%s %s\n", r.Hash, r.Name)
		}
	}
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
	_, _, err = backend.CopyFromStorerToRepo(ctx, pgStorer, repo)
	if err != nil {
		die("fetch: %v", err)
	}
	fmt.Println()
	os.Stdout.Sync()
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
	rest = strings.TrimPrefix(rest, "+")
	before, after, ok := strings.Cut(rest, ":")
	if !ok {
		return append(specs, pushSpec{dst: rest})
	}
	return append(specs, pushSpec{src: before, dst: after})
}

func readPushLines(scanner *bufio.Scanner, specs *[]pushSpec) {
	if *specs == nil {
		*specs = make([]pushSpec, 0, 16)
	}
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
	if _, err := backend.CopyObjectsFromRepoToStorer(repo, pgStorer); err != nil {
		die("iterate objects: %v", err)
	}
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
	_, err = q.RefsHasHEAD(ctx, repoID)
	if err != nil && len(specs) > 0 {
		_ = q.RefInsertSymbolic(ctx, db.RefInsertSymbolicParams{
			RepoID: repoID, Name: "HEAD", Symbolic: pgtype.Text{String: specs[0].dst, Valid: true},
		})
	}
	fmt.Println()
	os.Stdout.Sync()
}
