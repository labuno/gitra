package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
)

// ReconcileResult summarizes one reconcile run.
type ReconcileResult struct {
	Changed  int
	Health   domain.BindingHealth
	Failures []string
}

// Reconciler aligns applied git config with desired account state
// (baseline §12). It only touches managed keys.
type Reconciler struct {
	deps Deps
}

// NewReconciler wires a reconciler over the same ports as the binding service.
func NewReconciler(deps Deps) *Reconciler { return &Reconciler{deps: deps} }

// ReconcileBinding repairs one binding. A missing repository path is reported
// as HealthMissing and is not an error (baseline §13).
func (r *Reconciler) ReconcileBinding(ctx context.Context, id domain.BindingID) (ReconcileResult, error) {
	return r.reconcileBinding(ctx, id)
}

// ReconcileAccount propagates account changes to every bound repository.
// A single failing repository is reported, not fatal.
func (r *Reconciler) ReconcileAccount(ctx context.Context, id domain.AccountID) (ReconcileResult, error) {
	bindings, err := r.deps.Bindings.ListByAccount(ctx, id)
	if err != nil {
		return ReconcileResult{}, err
	}
	aggregate := ReconcileResult{Health: domain.HealthOK}
	for _, binding := range bindings {
		result, err := r.reconcileBinding(ctx, binding.ID)
		if err != nil {
			aggregate.Failures = append(aggregate.Failures, fmt.Sprintf("%s: %v", binding.Repository.Path, err))
			continue
		}
		aggregate.Changed += result.Changed
		if result.Health == domain.HealthMissing {
			aggregate.Failures = append(aggregate.Failures, fmt.Sprintf("%s: repository path not found", binding.Repository.Path))
		}
		for _, failure := range result.Failures {
			aggregate.Failures = append(aggregate.Failures, fmt.Sprintf("%s: %s", binding.Repository.Path, failure))
		}
	}
	if len(aggregate.Failures) > 0 {
		aggregate.Health = domain.HealthBroken
	}
	return aggregate, nil
}

func (r *Reconciler) reconcileBinding(ctx context.Context, id domain.BindingID) (ReconcileResult, error) {
	binding, err := r.deps.Bindings.Get(ctx, id)
	if err != nil {
		return ReconcileResult{}, err
	}
	account, err := r.deps.Accounts.Get(ctx, binding.AccountID)
	if err != nil {
		return ReconcileResult{}, err
	}
	strategy, err := r.deps.Routing.Get(domain.StrategyRepoLocal)
	if err != nil {
		return ReconcileResult{}, err
	}

	repo, err := r.deps.Git.DiscoverRepository(ctx, binding.Repository.Path)
	if err != nil {
		if errors.Is(err, domain.ErrNotGitRepository) {
			return ReconcileResult{Health: domain.HealthMissing}, nil
		}
		return ReconcileResult{}, err
	}

	changed := 0
	// Path move recovery (baseline §33): the repository metadata still
	// identifies the binding, so the central record follows the new path.
	if repo.RootPath != binding.Repository.Path {
		binding.Repository.Path = repo.RootPath
		binding.Revision++
		if err := r.deps.Bindings.Save(ctx, binding); err != nil {
			return ReconcileResult{}, err
		}
		changed++
	}

	entries, err := desiredEntries(ctx, r.deps, account, binding.ID, remoteExplicitPort(repo))
	if err != nil {
		return ReconcileResult{}, err
	}
	observed, err := strategy.Inspect(ctx, r.deps.Git, repo, entries)
	if err != nil {
		return ReconcileResult{}, err
	}
	if len(observed.Drift) > 0 {
		if err := strategy.Apply(ctx, r.deps.Git, repo, entries); err != nil {
			return ReconcileResult{}, err
		}
		changed += len(observed.Drift)
		observed, err = strategy.Inspect(ctx, r.deps.Git, repo, entries)
		if err != nil {
			return ReconcileResult{}, err
		}
		if len(observed.Drift) > 0 {
			return ReconcileResult{Changed: changed, Health: domain.HealthDrift, Failures: observed.Drift}, nil
		}
	}

	bindingID, accountID, present, err := readRepoMetadata(ctx, r.deps, repo)
	if err != nil {
		return ReconcileResult{}, err
	}
	matches := present && bindingID == string(binding.ID) && accountID == string(binding.AccountID)
	health, _ := computeHealth(healthInput{
		HasCentralBinding: true,
		RepoExists:        true,
		MetadataPresent:   present,
		MetadataMatches:   matches,
		Drift:             observed.Drift,
	})
	return ReconcileResult{Changed: changed, Health: health}, nil
}
