package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhanhd/gitra/internal/domain"
)

// CreateAccountRequest is the desired state for a new account.
type CreateAccountRequest struct {
	Alias     string
	Provider  domain.ProviderRef
	Identity  domain.CommitIdentity
	Transport domain.TransportConfig
}

// UpdateAccountRequest replaces the mutable account fields. ID and Revision
// are owned by the service.
type UpdateAccountRequest struct {
	Account domain.Account
}

// DeleteAccountRequest removes an account; UnbindAll is required when the
// account still has bound repositories (baseline §5.3).
type DeleteAccountRequest struct {
	ID        domain.AccountID
	UnbindAll bool
}

// AccountService owns the account lifecycle (baseline §5.1/§5.2/§5.3).
type AccountService struct {
	deps         Deps
	bindings     *BindingService
	reconciler   *Reconciler
	newAccountID func() (domain.AccountID, error)
}

// NewAccountService wires the account use cases.
func NewAccountService(deps Deps, bindings *BindingService) *AccountService {
	return &AccountService{
		deps:         deps,
		bindings:     bindings,
		reconciler:   NewReconciler(deps),
		newAccountID: generateAccountID,
	}
}

// Create validates and persists a new account. No network access happens here.
func (s *AccountService) Create(ctx context.Context, req CreateAccountRequest) (domain.Account, error) {
	var created domain.Account
	err := s.deps.Locker.WithWriteLock(ctx, func() error {
		id, err := s.newAccountID()
		if err != nil {
			return err
		}
		account := domain.Account{
			ID:        id,
			Alias:     req.Alias,
			Provider:  req.Provider,
			Identity:  req.Identity,
			Transport: req.Transport,
			Revision:  1,
		}
		if err := account.Validate(); err != nil {
			return err
		}
		if err := s.validateLocalAuth(ctx, account); err != nil {
			return err
		}
		if err := s.deps.Accounts.Save(ctx, account); err != nil {
			return err
		}
		created = account
		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}
	return created, nil
}

// Update saves the new account state, bumps Revision, then reconciles every
// bound repository. When reconciliation reports problems the account is still
// saved and the error wraps ErrStateDrift so callers can report both facts.
func (s *AccountService) Update(ctx context.Context, req UpdateAccountRequest) (domain.Account, error) {
	var updated domain.Account
	err := s.deps.Locker.WithWriteLock(ctx, func() error {
		existing, err := s.deps.Accounts.Get(ctx, req.Account.ID)
		if err != nil {
			return err
		}
		next := req.Account
		next.ID = existing.ID
		next.Revision = existing.Revision + 1
		if err := next.Validate(); err != nil {
			return err
		}
		if err := s.validateLocalAuth(ctx, next); err != nil {
			return err
		}
		if err := s.deps.Accounts.Save(ctx, next); err != nil {
			return err
		}
		updated = next
		return nil
	})
	if err != nil {
		return domain.Account{}, err
	}

	result, err := s.reconciler.ReconcileAccount(ctx, updated.ID)
	if err != nil {
		return updated, fmt.Errorf("%w: account saved but reconcile failed: %v", domain.ErrStateDrift, err)
	}
	if len(result.Failures) > 0 {
		return updated, fmt.Errorf("%w: account saved but %d repository operation(s) need attention: %v",
			domain.ErrStateDrift, len(result.Failures), result.Failures)
	}
	return updated, nil
}

// Delete removes an account. With bound repositories and UnbindAll=false it
// returns ErrAccountInUse; with UnbindAll=true each repository is unbound
// first (each unbind takes its own write lock).
func (s *AccountService) Delete(ctx context.Context, req DeleteAccountRequest) error {
	if _, err := s.deps.Accounts.Get(ctx, req.ID); err != nil {
		return err
	}
	bindings, err := s.deps.Bindings.ListByAccount(ctx, req.ID)
	if err != nil {
		return err
	}
	if len(bindings) > 0 && !req.UnbindAll {
		return fmt.Errorf("%w: %d bound repositories", domain.ErrAccountInUse, len(bindings))
	}
	for _, binding := range bindings {
		if err := s.bindings.Unbind(ctx, UnbindRequest{Path: binding.Repository.Path}); err != nil {
			return fmt.Errorf("unbind %s: %w", binding.Repository.Path, err)
		}
	}
	// Unbind takes the lock itself, so the final store write takes it here
	// rather than around the whole method (the locker is not re-entrant).
	return s.deps.Locker.WithWriteLock(ctx, func() error {
		return s.deps.Accounts.Delete(ctx, req.ID)
	})
}

// validateLocalAuth runs the offline half of account validation: the
// transport strategy checks local material (for ssh-key: the key file exists).
// No network access happens here.
func (s *AccountService) validateLocalAuth(ctx context.Context, account domain.Account) error {
	strategy, err := s.deps.Auth.Get(account.Transport.Strategy)
	if err != nil {
		return err
	}
	return strategy.Validate(ctx, account)
}

// Get returns one account by ID.
func (s *AccountService) Get(ctx context.Context, id domain.AccountID) (domain.Account, error) {
	return s.deps.Accounts.Get(ctx, id)
}

// GetByAlias resolves a user-supplied alias (or an acc_ identifier) to an
// account; the Application layer normalizes aliases early (baseline §4.1).
func (s *AccountService) GetByAlias(ctx context.Context, alias string) (domain.Account, error) {
	if strings.HasPrefix(alias, "acc_") {
		if account, err := s.deps.Accounts.Get(ctx, domain.AccountID(alias)); err == nil {
			return account, nil
		}
	}
	accounts, err := s.deps.Accounts.List(ctx)
	if err != nil {
		return domain.Account{}, err
	}
	for _, account := range accounts {
		if account.Alias == alias {
			return account, nil
		}
	}
	return domain.Account{}, fmt.Errorf("%w: %s", domain.ErrAccountNotFound, alias)
}

// List returns every account sorted by AccountID.
func (s *AccountService) List(ctx context.Context) ([]domain.Account, error) {
	return s.deps.Accounts.List(ctx)
}
