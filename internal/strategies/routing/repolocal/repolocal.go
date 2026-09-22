// Package repolocal implements the V1 repo-local routing strategy
// (baseline §19): managed keys are projected into the repository's own config.
package repolocal

import (
	"context"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/routing"
)

// ID is the strategy identifier stored in repo metadata and bindings.
const ID = "repo-local"

// Strategy writes managed entries into <repo>/.git/config via the Git port.
type Strategy struct{}

// New returns the repo-local strategy.
func New() *Strategy { return &Strategy{} }

// ID implements routing.Strategy.
func (s *Strategy) ID() string { return ID }

// Apply writes entries in order; callers own rollback on error.
func (s *Strategy) Apply(ctx context.Context, git ports.Git, repo domain.Repository, entries []ports.GitConfigEntry) error {
	for _, entry := range entries {
		if err := git.SetLocalConfig(ctx, repo, entry.Key, entry.Value); err != nil {
			return fmt.Errorf("apply %s: %w", entry.Key, err)
		}
	}
	return nil
}

// Inspect reads the entries back and reports every missing/different key.
func (s *Strategy) Inspect(ctx context.Context, git ports.Git, repo domain.Repository, entries []ports.GitConfigEntry) (routing.Status, error) {
	status := routing.Status{}
	for _, entry := range entries {
		value, found, err := git.GetLocalConfig(ctx, repo, entry.Key)
		if err != nil {
			return routing.Status{}, err
		}
		status.Actual = append(status.Actual, ports.GitConfigEntry{Key: entry.Key, Value: value})
		if !found || value != entry.Value {
			status.Drift = append(status.Drift, entry.Key)
		}
	}
	return status, nil
}

// Restore replays a snapshot: set previous values, unset keys that did not
// exist before binding.
func (s *Strategy) Restore(ctx context.Context, git ports.Git, repo domain.Repository, previous []routing.PreviousEntry) error {
	for _, entry := range previous {
		if entry.Exists {
			if err := git.SetLocalConfig(ctx, repo, entry.Key, entry.Value); err != nil {
				return fmt.Errorf("restore %s: %w", entry.Key, err)
			}
			continue
		}
		if err := git.UnsetLocalConfig(ctx, repo, entry.Key); err != nil {
			return fmt.Errorf("restore %s: %w", entry.Key, err)
		}
	}
	return nil
}
