package ports

import (
	"context"

	"github.com/zhanhd/gitra/internal/domain"
)

// Git is the outbound port to the native git executable (baseline §27).
// Implementations must not parse .git/config directly.
type Git interface {
	// DiscoverRepository resolves the repository containing path, returning
	// the canonical root plus the real git directory (never assumed to be
	// <root>/.git).
	DiscoverRepository(ctx context.Context, path string) (domain.Repository, error)

	// GetLocalConfig reads one repo-local key. found is false when unset.
	GetLocalConfig(ctx context.Context, repo domain.Repository, key string) (value string, found bool, err error)

	SetLocalConfig(ctx context.Context, repo domain.Repository, key string, value string) error

	UnsetLocalConfig(ctx context.Context, repo domain.Repository, key string) error

	Remotes(ctx context.Context, repo domain.Repository) ([]domain.Remote, error)
}

// GitConfigEntry is one repo-local git config key/value pair projected by a
// strategy (baseline §9 ownership whitelist).
type GitConfigEntry struct {
	Key   string
	Value string
}
