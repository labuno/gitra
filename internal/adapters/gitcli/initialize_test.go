package gitcli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/runner"
)

func TestInitializeExistingFolderWithoutCommits(t *testing.T) {
	path := t.TempDir()
	if err := os.WriteFile(filepath.Join(path, "code.txt"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := New(runner.New())
	if err := adapter.InitializeRepository(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.DiscoverRepository(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(mustRun(t, path, "symbolic-ref", "--short", "HEAD")); got != "main" {
		t.Fatalf("branch = %s", got)
	}
	if got := strings.TrimSpace(mustRun(t, path, "status", "--porcelain")); got != "?? code.txt" {
		t.Fatalf("initialization should leave code unstaged: %s", got)
	}
}

func TestInitializeRejectsMissingFolderAndFile(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(base, "missing"), file} {
		if err := New(runner.New()).InitializeRepository(context.Background(), path); err == nil {
			t.Fatalf("must reject %s", path)
		}
	}
	if _, err := os.Stat(filepath.Join(base, "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing directory must not be created: %v", err)
	}
}
