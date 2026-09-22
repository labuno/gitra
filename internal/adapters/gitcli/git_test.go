package gitcli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/domain"
)

func mustRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func newRepo(t *testing.T, remoteURL string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "luna-site")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	mustRun(t, root, "init", "-q")
	mustRun(t, root, "config", "user.name", "Old Name")
	mustRun(t, root, "config", "user.email", "old@example.com")
	if remoteURL != "" {
		mustRun(t, root, "remote", "add", "origin", remoteURL)
	}
	return root
}

func TestDiscoverRepository(t *testing.T) {
	root := newRepo(t, "git@github.com:lunafoundry/luna-site.git")
	sub := filepath.Join(root, "internal", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := New(runner.New())
	repo, err := adapter.DiscoverRepository(context.Background(), sub)
	if err != nil {
		t.Fatalf("DiscoverRepository() error = %v", err)
	}
	realRoot, _ := filepath.EvalSymlinks(root)
	if repo.RootPath != realRoot {
		t.Fatalf("RootPath = %q, want %q", repo.RootPath, realRoot)
	}
	if repo.GitDir != filepath.Join(realRoot, ".git") {
		t.Fatalf("GitDir = %q", repo.GitDir)
	}
	if len(repo.Remotes) != 1 || repo.Remotes[0].Name != "origin" {
		t.Fatalf("Remotes = %+v", repo.Remotes)
	}
	if err := repo.Validate(); err != nil {
		t.Fatalf("discovered repository invalid: %v", err)
	}
}

func TestDiscoverRepositoryErrors(t *testing.T) {
	adapter := New(runner.New())
	if _, err := adapter.DiscoverRepository(context.Background(), t.TempDir()); !errors.Is(err, domain.ErrNotGitRepository) {
		t.Fatalf("non-repo error = %v, want ErrNotGitRepository", err)
	}

	bare := filepath.Join(t.TempDir(), "bare.git")
	mustRun(t, filepath.Dir(bare), "init", "-q", "--bare", bare)
	if _, err := adapter.DiscoverRepository(context.Background(), bare); !errors.Is(err, domain.ErrUnsupportedRepo) {
		t.Fatalf("bare error = %v, want ErrUnsupportedRepo", err)
	}

	main := newRepo(t, "")
	worktree := filepath.Join(t.TempDir(), "wt")
	mustRun(t, main, "worktree", "add", "-q", worktree)
	if _, err := adapter.DiscoverRepository(context.Background(), worktree); !errors.Is(err, domain.ErrUnsupportedRepo) {
		t.Fatalf("worktree error = %v, want ErrUnsupportedRepo", err)
	}
}

func TestLocalConfigRoundTrip(t *testing.T) {
	root := newRepo(t, "git@github.com:lunafoundry/luna-site.git")
	adapter := New(runner.New())
	repo, err := adapter.DiscoverRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if value, found, err := adapter.GetLocalConfig(ctx, repo, "core.sshCommand"); err != nil || found || value != "" {
		t.Fatalf("unset key = (%q, %v, %v), want (\"\", false, nil)", value, found, err)
	}
	if err := adapter.SetLocalConfig(ctx, repo, "core.sshCommand", `ssh -i "/tmp/key" -o IdentitiesOnly=yes`); err != nil {
		t.Fatal(err)
	}
	value, found, err := adapter.GetLocalConfig(ctx, repo, "core.sshCommand")
	if err != nil || !found || value != `ssh -i "/tmp/key" -o IdentitiesOnly=yes` {
		t.Fatalf("get = (%q, %v, %v)", value, found, err)
	}
	if err := adapter.UnsetLocalConfig(ctx, repo, "core.sshCommand"); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := adapter.GetLocalConfig(ctx, repo, "core.sshCommand"); found {
		t.Fatal("key still present after unset")
	}
	// unsetting a missing key stays idempotent
	if err := adapter.UnsetLocalConfig(ctx, repo, "core.sshCommand"); err != nil {
		t.Fatalf("second unset error = %v", err)
	}
}

func TestGetGlobalConfig(t *testing.T) {
	adapter := New(runner.New())
	// A missing key is (_, false, nil); the test machine may or may not have one.
	if value, found, err := adapter.GetGlobalConfig(context.Background(), "gitra.test.nonexistent"); err != nil || found || value != "" {
		t.Fatalf("missing global key = (%q, %v, %v)", value, found, err)
	}
}

func TestRemotes(t *testing.T) {
	root := newRepo(t, "git@github.com:lunafoundry/luna-site.git")
	mustRun(t, root, "remote", "add", "upstream", "ssh://git@git.example.com:2222/luna/repo.git")
	adapter := New(runner.New())
	repo, err := adapter.DiscoverRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Remotes) != 2 {
		t.Fatalf("Remotes = %+v", repo.Remotes)
	}
	parsed, err := domain.ParseRemoteURL("ssh://git@git.example.com:2222/luna/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "git.example.com" || parsed.Port != 2222 || !parsed.ExplicitPort {
		t.Fatalf("parsed = %+v", parsed)
	}
}
