//go:build integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteHelper_Capabilities(t *testing.T) {
	connStr := os.Getenv("PGCONN")
	if connStr == "" {
		t.Skip("PGCONN not set (integration only)")
	}

	bin := buildBackend(t)
	url := connStr + "/remote_helper_cap_test"
	cmd := exec.Command(bin, "pg", url)
	cmd.Stdin = strings.NewReader("capabilities\n\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("remote helper: %v\n%s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "fetch") || !strings.Contains(s, "push") {
		t.Errorf("capabilities output missing fetch/push: %q", s)
	}
}

func TestRemoteHelper_List(t *testing.T) {
	connStr := os.Getenv("PGCONN")
	if connStr == "" {
		t.Skip("PGCONN not set (integration only)")
	}

	bin := buildBackend(t)
	repoName := "remote_helper_list_test"
	initCmd := exec.Command(bin, "init", connStr, repoName)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	url := connStr + "/" + repoName
	cmd := exec.Command(bin, "pg", url)
	cmd.Stdin = strings.NewReader("list\n\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("remote helper list: %v\n%s", err, out)
	}
	// Empty repo: output is blank line only (cmdList prints refs then "\n")
	_ = string(out)
}

func buildBackend(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "gitgres-backend")
	if os.PathSeparator == '\\' {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/backend")
	cmd.Dir = repoRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
