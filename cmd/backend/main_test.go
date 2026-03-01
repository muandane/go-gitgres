package main

import (
	"bufio"
	"context"
	"os"
	"strings"
	"testing"

	"go-gitgres/internal/db"
)

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

	connStr := os.Getenv("PGCONN")
	if connStr == "" {
		connStr = "dbname=" + os.Getenv("PGDATABASE")
		if connStr == "dbname=" {
			connStr = "dbname=gitgres_test"
		}
	}

	runInit(ctx, connStr, "go_backend_init_test")
	// Idempotent
	runInit(ctx, connStr, "go_backend_init_test")
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

	connStr := os.Getenv("PGCONN")
	if connStr == "" {
		connStr = "dbname=gitgres_test"
	}

	// Ensure repo exists
	runInit(ctx, connStr, "go_backend_lsrefs_test")
	runLsRefs(ctx, connStr, "go_backend_lsrefs_test")
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
