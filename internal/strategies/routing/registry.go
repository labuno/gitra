// Package routing holds pluggable repository routing strategies and their
// registry (baseline §19). V1 ships repo-local only.
package routing

import (
	"context"
	"errors"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// ErrUnknownStrategy reports an unregistered strategy ID.
var ErrUnknownStrategy = errors.New("unknown strategy")

// PreviousEntry is one pre-bind config value used by Restore, mirroring the
// repository snapshot schema (baseline §11).
type PreviousEntry struct {
	Key    string
	Exists bool
	Value  string
}

// Status is the observed state of the entries managed by a routing strategy.
type Status struct {
	Actual []ports.GitConfigEntry
	Drift  []string
}

// Strategy projects managed entries into a repository and inspects them.
type Strategy interface {
	ID() string
	Apply(ctx context.Context, git ports.Git, repo domain.Repository, entries []ports.GitConfigEntry) error
	Inspect(ctx context.Context, git ports.Git, repo domain.Repository, entries []ports.GitConfigEntry) (Status, error)
	Restore(ctx context.Context, git ports.Git, repo domain.Repository, previous []PreviousEntry) error
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
		return fmt.Errorf("routing strategy %q already registered", strategy.ID())
	}
	r.strategies[strategy.ID()] = strategy
	return nil
}

// Get returns the strategy for id.
func (r *Registry) Get(id string) (Strategy, error) {
	strategy, ok := r.strategies[id]
	if !ok {
		return nil, fmt.Errorf("%w: routing %s", ErrUnknownStrategy, id)
	}
	return strategy, nil
}
