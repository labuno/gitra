package ports

import (
	"context"

	"github.com/zhanhd/gitra/internal/domain"
)

// AccountStore persists desired account state (baseline §16.3).
type AccountStore interface {
	Save(ctx context.Context, account domain.Account) error
	Get(ctx context.Context, id domain.AccountID) (domain.Account, error)
	List(ctx context.Context) ([]domain.Account, error)
	Delete(ctx context.Context, id domain.AccountID) error
}

// BindingStore persists central binding records (baseline §16.4).
type BindingStore interface {
	Save(ctx context.Context, binding domain.RepositoryBinding) error
	Get(ctx context.Context, id domain.BindingID) (domain.RepositoryBinding, error)
	// FindByRepository looks up a binding by canonical repository path; the
	// boolean is false when the path is not bound.
	FindByRepository(ctx context.Context, canonicalPath string) (domain.RepositoryBinding, bool, error)
	ListByAccount(ctx context.Context, id domain.AccountID) ([]domain.RepositoryBinding, error)
	Delete(ctx context.Context, id domain.BindingID) error
}
