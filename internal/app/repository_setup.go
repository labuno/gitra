package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
)

// PrepareRepository leaves existing repositories intact and initializes plain folders.
func (s *BindingService) PrepareRepository(ctx context.Context, accountID domain.AccountID, path string) (domain.Repository, error) {
	account, err := s.deps.Accounts.Get(ctx, accountID)
	if err != nil {
		return domain.Repository{}, err
	}
	strategy, err := s.deps.Auth.Get(account.Transport.Strategy)
	if err != nil {
		return domain.Repository{}, err
	}
	if err := strategy.Validate(ctx, account); err != nil {
		return domain.Repository{}, err
	}
	var repo domain.Repository
	err = s.deps.Locker.WithWriteLock(ctx, func() error {
		var prepareErr error
		repo, prepareErr = s.prepareRepositoryLocked(ctx, path)
		return prepareErr
	})
	return repo, err
}

func (s *BindingService) prepareRepositoryLocked(ctx context.Context, path string) (domain.Repository, error) {
	repo, err := s.deps.Git.DiscoverRepository(ctx, path)
	if !errors.Is(err, domain.ErrNotGitRepository) {
		return repo, err
	}
	if s.deps.Initializer == nil {
		return domain.Repository{}, fmt.Errorf("repository initialization is unavailable")
	}
	if err := s.deps.Initializer.InitializeRepository(ctx, path); err != nil {
		return domain.Repository{}, err
	}
	return s.deps.Git.DiscoverRepository(ctx, path)
}
