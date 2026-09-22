// Package gitcli implements the Git port by invoking the native git
// executable. It never parses .git/config directly (baseline §17.1).
package gitcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// Adapter implements ports.Git.
type Adapter struct {
	runner ports.CommandRunner
}

// New builds an Adapter around a process runner.
func New(runner ports.CommandRunner) *Adapter { return &Adapter{runner: runner} }

func (a *Adapter) run(ctx context.Context, dir string, args ...string) (ports.ProcessResult, error) {
	full := append([]string{"-C", dir}, args...)
	return a.runner.Run(ctx, "git", full...)
}

// DiscoverRepository resolves the repository containing path using git itself
// (baseline §4.5/§4.9): canonical root, real git dir, bare and linked-worktree
// rejection.
func (a *Adapter) DiscoverRepository(ctx context.Context, path string) (domain.Repository, error) {
	dir, err := expandUser(path)
	if err != nil {
		return domain.Repository{}, err
	}

	bare, err := a.run(ctx, dir, "rev-parse", "--is-bare-repository")
	if err != nil {
		return domain.Repository{}, err
	}
	if bare.ExitCode != 0 {
		return domain.Repository{}, fmt.Errorf("%w: %s", domain.ErrNotGitRepository, firstLine(bare.Stderr))
	}
	if strings.TrimSpace(bare.Stdout) == "true" {
		return domain.Repository{}, fmt.Errorf("%w: bare repositories are not supported in V1", domain.ErrUnsupportedRepo)
	}

	top, err := a.run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return domain.Repository{}, err
	}
	if top.ExitCode != 0 {
		return domain.Repository{}, fmt.Errorf("%w: %s", domain.ErrNotGitRepository, firstLine(top.Stderr))
	}
	root := strings.TrimSpace(top.Stdout)

	gitDirRes, err := a.run(ctx, root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return domain.Repository{}, err
	}
	if gitDirRes.ExitCode != 0 {
		return domain.Repository{}, fmt.Errorf("%w: %s", domain.ErrNotGitRepository, firstLine(gitDirRes.Stderr))
	}
	gitDir := strings.TrimSpace(gitDirRes.Stdout)

	commonRes, err := a.run(ctx, root, "rev-parse", "--git-common-dir")
	if err != nil {
		return domain.Repository{}, err
	}
	if commonRes.ExitCode == 0 {
		common := strings.TrimSpace(commonRes.Stdout)
		if !filepath.IsAbs(common) {
			common = filepath.Join(root, common)
		}
		if resolved, err := filepath.EvalSymlinks(common); err == nil {
			common = resolved
		}
		if common != gitDir {
			return domain.Repository{}, fmt.Errorf("%w: linked worktrees are not supported in V1", domain.ErrUnsupportedRepo)
		}
	}

	repo := domain.Repository{RootPath: root, GitDir: gitDir}
	remotes, err := a.Remotes(ctx, repo)
	if err != nil {
		return domain.Repository{}, err
	}
	repo.Remotes = remotes
	return repo, nil
}

// GetLocalConfig reads a repo-local key; found is false when unset.
func (a *Adapter) GetLocalConfig(ctx context.Context, repo domain.Repository, key string) (string, bool, error) {
	res, err := a.run(ctx, repo.RootPath, "config", "--local", "--get", key)
	if err != nil {
		return "", false, err
	}
	if res.ExitCode == 1 {
		return "", false, nil
	}
	if res.ExitCode != 0 {
		return "", false, fmt.Errorf("git config --get %s: %s", key, firstLine(res.Stderr))
	}
	return strings.TrimRight(res.Stdout, "\n"), true, nil
}

// SetLocalConfig writes one repo-local key.
func (a *Adapter) SetLocalConfig(ctx context.Context, repo domain.Repository, key, value string) error {
	res, err := a.run(ctx, repo.RootPath, "config", "--local", key, value)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("git config %s: %s", key, firstLine(res.Stderr))
	}
	return nil
}

// UnsetLocalConfig removes a repo-local key; unsetting an absent key is a
// no-op so callers stay idempotent.
func (a *Adapter) UnsetLocalConfig(ctx context.Context, repo domain.Repository, key string) error {
	res, err := a.run(ctx, repo.RootPath, "config", "--local", "--unset", key)
	if err != nil {
		return err
	}
	if res.ExitCode == 0 || res.ExitCode == 5 {
		return nil
	}
	return fmt.Errorf("git config --unset %s: %s", key, firstLine(res.Stderr))
}

// Remotes lists configured remotes in deterministic order.
func (a *Adapter) Remotes(ctx context.Context, repo domain.Repository) ([]domain.Remote, error) {
	res, err := a.run(ctx, repo.RootPath, "config", "--get-regexp", `^remote\..*\.url$`)
	if err != nil {
		return nil, err
	}
	if res.ExitCode == 1 {
		return nil, nil
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("git config --get-regexp remotes: %s", firstLine(res.Stderr))
	}
	var remotes []domain.Remote
	for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(key, "remote."), ".url")
		remotes = append(remotes, domain.Remote{Name: name, URL: strings.TrimSpace(value)})
	}
	sort.Slice(remotes, func(i, j int) bool { return remotes[i].Name < remotes[j].Name })
	return remotes, nil
}

func expandUser(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	if path == "" {
		return "", fmt.Errorf("empty repository path")
	}
	return path, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
