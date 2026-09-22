// Package auth holds pluggable transport-authentication strategies and their
// registry (baseline §18). V1 ships ssh-key only; HTTPS token is V1.1.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// ErrUnknownStrategy reports an unregistered strategy ID.
var ErrUnknownStrategy = errors.New("unknown strategy")

// BuildRequest carries everything needed to build repo-local auth config.
type BuildRequest struct {
	Account domain.Account
	// RemoteHasExplicitPort is true when the selected remote URL carries its
	// own port; baseline §14 then forbids appending -p to core.sshCommand.
	RemoteHasExplicitPort bool
}

// Strategy turns an account into git config entries for transport auth.
type Strategy interface {
	ID() string
	Validate(ctx context.Context, account domain.Account) error
	BuildGitConfig(req BuildRequest) ([]ports.GitConfigEntry, error)
}

// Registry maps strategy IDs to implementations.
type Registry struct {
	strategies map[string]Strategy
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{strategies: map[string]Strategy{}}
}

// Register adds one strategy; duplicate IDs are rejected.
func (r *Registry) Register(strategy Strategy) error {
	if _, exists := r.strategies[strategy.ID()]; exists {
		return fmt.Errorf("auth strategy %q already registered", strategy.ID())
	}
	r.strategies[strategy.ID()] = strategy
	return nil
}

// Get returns the strategy for id.
func (r *Registry) Get(id string) (Strategy, error) {
	strategy, ok := r.strategies[id]
	if !ok {
		return nil, fmt.Errorf("%w: auth %s", ErrUnknownStrategy, id)
	}
	return strategy, nil
}
