package domain

import (
	"path/filepath"
	"strings"
)

// Remote is one named remote of a repository.
type Remote struct {
	Name string
	URL  string
}

// Repository is the runtime view of a non-bare repository. GitDir is resolved
// through git rev-parse and never assumed to be <root>/.git (baseline §4.5).
type Repository struct {
	RootPath string
	GitDir   string
	Remotes  []Remote
}

// Validate checks structural invariants only; discovery belongs to the adapter.
func (r Repository) Validate() error {
	if strings.TrimSpace(r.RootPath) == "" {
		return invalid("root_path", "must not be empty")
	}
	if strings.TrimSpace(r.GitDir) == "" {
		return invalid("git_dir", "must not be empty")
	}
	seen := make(map[string]bool, len(r.Remotes))
	for _, remote := range r.Remotes {
		if strings.TrimSpace(remote.Name) == "" || strings.TrimSpace(remote.URL) == "" {
			return invalid("remotes", "remote name and url must not be empty")
		}
		if seen[remote.Name] {
			return invalid("remotes", "remote names must be unique")
		}
		seen[remote.Name] = true
	}
	return nil
}

// RemoteByName returns the named remote when present.
func (r Repository) RemoteByName(name string) (Remote, bool) {
	for _, remote := range r.Remotes {
		if remote.Name == name {
			return remote, true
		}
	}
	return Remote{}, false
}

// RepositoryRef points at a canonical repository root (baseline §4.9).
type RepositoryRef struct {
	Path string
}

func (r RepositoryRef) Validate() error {
	path := strings.TrimSpace(r.Path)
	if path == "" {
		return invalid("repository.path", "must not be empty")
	}
	if !filepath.IsAbs(path) {
		return invalid("repository.path", "must be an absolute canonical path")
	}
	return nil
}
