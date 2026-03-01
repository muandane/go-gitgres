//go:build integration

package testutil

import (
	"context"
	"testing"
)

func TestStartPostgres(t *testing.T) {
	ctx := context.Background()
	connStr, cleanup, err := StartPostgres(ctx, nil)
	if err != nil {
		t.Fatalf("StartPostgres: %v", err)
	}
	defer cleanup()
	if connStr == "" {
		t.Error("StartPostgres: connStr empty")
	}
}

func TestFindModuleRoot(t *testing.T) {
	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("findModuleRoot: %v", err)
	}
	if root == "" {
		t.Error("findModuleRoot: root empty")
	}
}
