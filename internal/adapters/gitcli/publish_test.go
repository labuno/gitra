package gitcli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/runner"
)

// TestFirstPublishAgainstEmptyRemote exercises the real publish path: an empty
// bare remote, a local commit, then push + upstream.
func TestFirstPublishAgainstEmptyRemote(t *testing.T) {
	base := t.TempDir()
	bare := filepath.Join(base, "remote.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	mustRun(t, filepath.Dir(bare), "init", "-q", "--bare", bare)

	repoPath := filepath.Join(base, "work")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatal(err)
	}
	mustRun(t, repoPath, "init", "-q", "-b", "main")
	mustRun(t, repoPath, "config", "user.name", "Luna")
	mustRun(t, repoPath, "config", "user.email", "luna@example.com")
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, repoPath, "add", ".")
	mustRun(t, repoPath, "commit", "-q", "-m", "initial")
	mustRun(t, repoPath, "remote", "add", "origin", bare)

	adapter := New(runner.New())
	ctx := context.Background()
	repo, err := adapter.DiscoverRepository(ctx, repoPath)
	if err != nil {
		t.Fatal(err)
	}

	status, err := adapter.PublishStatus(ctx, repo)
	if err != nil {
		t.Fatalf("PublishStatus() error = %v", err)
	}
	if status.Branch != "main" || !status.HasLocalCommits || status.RemoteHasBranch {
		t.Fatalf("status = %+v", status)
	}

	if err := adapter.Push(ctx, repo, "origin", "main", true); err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	status, err = adapter.PublishStatus(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !status.RemoteHasBranch {
		t.Fatal("remote branch missing after push")
	}
}

func TestPublishStatusReportsNoCommits(t *testing.T) {
	repoPath := newRepo(t, "")
	adapter := New(runner.New())
	repo, err := adapter.DiscoverRepository(context.Background(), repoPath)
	if err != nil {
		t.Fatal(err)
	}
	// A fresh repository has an unborn branch: symbolic-ref still resolves.
	status, err := adapter.PublishStatus(context.Background(), repo)
	if err != nil {
		t.Fatalf("PublishStatus() error = %v", err)
	}
	if status.HasLocalCommits {
		t.Fatalf("status = %+v, want no commits", status)
	}
	if status.Branch == "" {
		t.Fatal("branch must be reported even without commits")
	}
}
