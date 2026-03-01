package main

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-gitgres/internal/db"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// exitSentinel is the value panic'd by tests when die would have called os.Exit(1).
type exitSentinel int

func TestRunCLI_NoArgs(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFunc
	oldStderr := os.Stderr
	defer func() { os.Args = oldArgs; exitFunc = oldExit; os.Stderr = oldStderr }()

	os.Args = []string{"gitgres-backend"}
	exitFunc = func(int) { panic(exitSentinel(1)) }
	r, w, _ := os.Pipe()
	os.Stderr = w

	var gotPanic bool
	func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(exitSentinel); ok {
					gotPanic = true
				} else {
					panic(v)
				}
			}
		}()
		runCLI()
	}()
	w.Close()
	var stderr bytes.Buffer
	stderr.ReadFrom(r)
	if !gotPanic {
		t.Fatal("runCLI did not exit (panic)")
	}
	want := "fatal: usage: gitgres-backend init|push|clone|ls-refs [args...]"
	if got := strings.TrimSpace(stderr.String()); got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestRunCLI_UnknownCommand(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFunc
	oldStderr := os.Stderr
	defer func() { os.Args = oldArgs; exitFunc = oldExit; os.Stderr = oldStderr }()

	os.Args = []string{"gitgres-backend", "unknown"}
	exitFunc = func(int) { panic(exitSentinel(1)) }
	r, w, _ := os.Pipe()
	os.Stderr = w

	var gotPanic bool
	func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(exitSentinel); ok {
					gotPanic = true
				} else {
					panic(v)
				}
			}
		}()
		runCLI()
	}()
	w.Close()
	var stderr bytes.Buffer
	stderr.ReadFrom(r)
	if !gotPanic {
		t.Fatal("runCLI did not exit (panic)")
	}
	want := "fatal: unknown command: unknown"
	if got := strings.TrimSpace(stderr.String()); got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestRunCLI_InitTooFewArgs(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFunc
	oldStderr := os.Stderr
	defer func() { os.Args = oldArgs; exitFunc = oldExit; os.Stderr = oldStderr }()

	os.Args = []string{"gitgres-backend", "init", "conn"}
	exitFunc = func(int) { panic(exitSentinel(1)) }
	r, w, _ := os.Pipe()
	os.Stderr = w

	var gotPanic bool
	func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(exitSentinel); ok {
					gotPanic = true
				} else {
					panic(v)
				}
			}
		}()
		runCLI()
	}()
	w.Close()
	var stderr bytes.Buffer
	stderr.ReadFrom(r)
	if !gotPanic {
		t.Fatal("runCLI did not exit (panic)")
	}
	want := "fatal: usage: gitgres-backend init <conninfo> <reponame>"
	if got := strings.TrimSpace(stderr.String()); got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestRunCLI_PushTooFewArgs(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFunc
	oldStderr := os.Stderr
	defer func() { os.Args = oldArgs; exitFunc = oldExit; os.Stderr = oldStderr }()

	os.Args = []string{"gitgres-backend", "push", "conn", "repo"}
	exitFunc = func(int) { panic(exitSentinel(1)) }
	r, w, _ := os.Pipe()
	os.Stderr = w
	var gotPanic bool
	func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(exitSentinel); ok {
					gotPanic = true
				} else {
					panic(v)
				}
			}
		}()
		runCLI()
	}()
	w.Close()
	var stderr bytes.Buffer
	stderr.ReadFrom(r)
	if !gotPanic {
		t.Fatal("runCLI did not exit (panic)")
	}
	want := "fatal: usage: gitgres-backend push <conninfo> <reponame> <local-repo-path>"
	if got := strings.TrimSpace(stderr.String()); got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestRunCLI_CloneTooFewArgs(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFunc
	oldStderr := os.Stderr
	defer func() { os.Args = oldArgs; exitFunc = oldExit; os.Stderr = oldStderr }()

	os.Args = []string{"gitgres-backend", "clone", "conn", "repo"}
	exitFunc = func(int) { panic(exitSentinel(1)) }
	r, w, _ := os.Pipe()
	os.Stderr = w
	var gotPanic bool
	func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(exitSentinel); ok {
					gotPanic = true
				} else {
					panic(v)
				}
			}
		}()
		runCLI()
	}()
	w.Close()
	var stderr bytes.Buffer
	stderr.ReadFrom(r)
	if !gotPanic {
		t.Fatal("runCLI did not exit (panic)")
	}
	want := "fatal: usage: gitgres-backend clone <conninfo> <reponame> <dest-dir>"
	if got := strings.TrimSpace(stderr.String()); got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func TestRunCLI_LsRefsTooFewArgs(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFunc
	oldStderr := os.Stderr
	defer func() { os.Args = oldArgs; exitFunc = oldExit; os.Stderr = oldStderr }()

	os.Args = []string{"gitgres-backend", "ls-refs", "conn"}
	exitFunc = func(int) { panic(exitSentinel(1)) }
	r, w, _ := os.Pipe()
	os.Stderr = w
	var gotPanic bool
	func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(exitSentinel); ok {
					gotPanic = true
				} else {
					panic(v)
				}
			}
		}()
		runCLI()
	}()
	w.Close()
	var stderr bytes.Buffer
	stderr.ReadFrom(r)
	if !gotPanic {
		t.Fatal("runCLI did not exit (panic)")
	}
	want := "fatal: usage: gitgres-backend ls-refs <conninfo> <reponame>"
	if got := strings.TrimSpace(stderr.String()); got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

func connStr(t *testing.T) string {
	t.Helper()
	s := os.Getenv("PGCONN")
	if s == "" {
		s = "dbname=gitgres_test"
	}
	return s
}

func TestRunInit(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	runInit(ctx, connStr(t), "go_backend_init_test")
	runInit(ctx, connStr(t), "go_backend_init_test")
}

func TestRunLsRefs(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}
	runInit(ctx, connStr(t), "go_backend_lsrefs_test")
	runLsRefs(ctx, connStr(t), "go_backend_lsrefs_test")
}

func TestRunPush(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
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
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := w.Add("f.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := w.Commit("first", &git.CommitOptions{Author: &object.Signature{Name: "t", Email: "t@t"}}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	runInit(ctx, connStr(t), "go_backend_push_test")
	runPush(ctx, connStr(t), "go_backend_push_test", dir)
}

func TestRunClone(t *testing.T) {
	ctx := context.Background()
	pool, err := db.OpenPool(ctx, "")
	if err != nil {
		t.Skipf("no DB: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	runInit(ctx, connStr(t), "go_backend_clone_test")
	pushDir := t.TempDir()
	repo, err := git.PlainInit(pushDir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pushDir, "f.txt"), []byte("y"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := w.Add("f.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := w.Commit("first", &git.CommitOptions{Author: &object.Signature{Name: "t", Email: "t@t"}}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	runPush(ctx, connStr(t), "go_backend_clone_test", pushDir)

	dest := t.TempDir()
	runClone(ctx, connStr(t), "go_backend_clone_test", dest)
	cloned, err := git.PlainOpen(dest)
	if err != nil {
		t.Fatalf("PlainOpen clone: %v", err)
	}
	refs, err := cloned.References()
	if err != nil {
		t.Fatalf("References: %v", err)
	}
	var count int
	_ = refs.ForEach(func(*plumbing.Reference) error { count++; return nil })
	refs.Close()
	if count == 0 {
		t.Error("clone has no refs")
	}
}

func TestParsePushLine(t *testing.T) {
	tests := []struct {
		line string
		want []pushSpec
	}{
		{"push refs/heads/main:refs/heads/main", []pushSpec{{src: "refs/heads/main", dst: "refs/heads/main"}}},
		{"push +refs/heads/main:refs/heads/main", []pushSpec{{src: "refs/heads/main", dst: "refs/heads/main"}}},
		{"push :refs/heads/delete", []pushSpec{{dst: "refs/heads/delete"}}},
		{"push branch", []pushSpec{{dst: "branch"}}},
		{"list", nil},
		{"", nil},
	}
	for _, tt := range tests {
		got := parsePushLine(tt.line)
		if len(got) != len(tt.want) {
			t.Errorf("parsePushLine(%q) len = %d, want %d", tt.line, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i].src != tt.want[i].src || got[i].dst != tt.want[i].dst {
				t.Errorf("parsePushLine(%q)[%d] = %+v, want %+v", tt.line, i, got[i], tt.want[i])
			}
		}
	}
}

func TestReadPushLines(t *testing.T) {
	input := "push refs/heads/a:refs/heads/a\npush refs/heads/b:refs/heads/b\n\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	var specs []pushSpec
	readPushLines(scanner, &specs)
	if len(specs) != 2 {
		t.Fatalf("readPushLines: got %d specs, want 2", len(specs))
	}
	if specs[0].src != "refs/heads/a" || specs[0].dst != "refs/heads/a" {
		t.Errorf("specs[0] = %+v", specs[0])
	}
	if specs[1].src != "refs/heads/b" || specs[1].dst != "refs/heads/b" {
		t.Errorf("specs[1] = %+v", specs[1])
	}
}

func TestReadFetchLines(t *testing.T) {
	input := "line1\nline2\n\nrest\nmore"
	scanner := bufio.NewScanner(strings.NewReader(input))
	readFetchLines(scanner)
	if !scanner.Scan() {
		t.Fatal("scanner should have next line after blank")
	}
	if got := scanner.Text(); got != "rest" {
		t.Errorf("after readFetchLines: next line = %q, want %q", got, "rest")
	}
	if !scanner.Scan() {
		t.Fatal("scanner should have one more line")
	}
	if got := scanner.Text(); got != "more" {
		t.Errorf("next line = %q, want %q", got, "more")
	}
}
