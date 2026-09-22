package ports

import (
	"context"

	"github.com/zhanhd/gitra/internal/domain"
)

// RemoteStatus is the provider-side state of a repository address.
type RemoteStatus struct {
	Exists   bool
	Private  bool
	CloneURL string
}

// RepoService performs provider-side repository operations with an account's
// stored credential (lookup before binding, create when publishing a local
// repository for the first time).
type RepoService interface {
	LookupRepo(ctx context.Context, providerType domain.ProviderType, host, token, owner, repo string) (RemoteStatus, error)
	CreateRepo(ctx context.Context, providerType domain.ProviderType, host, token, owner, name string, private bool) (RemoteStatus, error)
}
