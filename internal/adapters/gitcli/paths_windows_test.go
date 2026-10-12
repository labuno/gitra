//go:build windows

package gitcli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/runner"
)

func TestWindowsGitDiscoveryNormalizesPaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "项目 with spaces")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := New(runner.New())
	if err := adapter.InitializeRepository(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	repo, err := adapter.DiscoverRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if repo.RootPath != root || repo.GitDir != filepath.Join(root, ".git") {
		t.Fatalf("native paths = %+v", repo)
	}
}
