package app

import (
	"context"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func TestReconcileBindingRepairsDriftAndIsIdempotent(t *testing.T) {
	service, git, _, _, account := newService(t, "git@github.com:lunafoundry/luna-site.git")
	ctx := context.Background()
	binding, err := service.Bind(ctx, BindRequest{AccountID: account.ID, Path: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	reconciler := NewReconciler(service.deps)

	git.values["user.email"] = "drifted@example.com"
	result, err := reconciler.ReconcileBinding(ctx, binding.ID)
	if err != nil {
		t.Fatalf("ReconcileBinding() error = %v", err)
	}
	if result.Health != domain.HealthOK || result.Changed == 0 {
		t.Fatalf("result = %+v", result)
	}
	if git.values["user.email"] != "luna@example.com" {
		t.Fatalf("drift not repaired: %q", git.values["user.email"])
	}

	result, err = reconciler.ReconcileBinding(ctx, binding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed != 0 || result.Health != domain.HealthOK {
		t.Fatalf("second run = %+v, want idempotent", result)
	}
}

func TestReconcileBindingMissingRepository(t *testing.T) {
	service, git, _, _, account := newService(t, "git@github.com:lunafoundry/luna-site.git")
	ctx := context.Background()
	binding, err := service.Bind(ctx, BindRequest{AccountID: account.ID, Path: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	git.missing["/repo"] = true
	result, err := NewReconciler(service.deps).ReconcileBinding(ctx, binding.ID)
	if err != nil {
		t.Fatalf("missing repository must not be an error, got %v", err)
	}
	if result.Health != domain.HealthMissing {
		t.Fatalf("health = %q, want missing", result.Health)
	}
}

func TestReconcileAccountPropagatesAndIsolatesFailures(t *testing.T) {
	service, git, bindings, _, account := newService(t, "git@github.com:lunafoundry/luna-site.git")
	ctx := context.Background()
	bound, err := service.Bind(ctx, BindRequest{AccountID: account.ID, Path: "/repo-a"})
	if err != nil {
		t.Fatal(err)
	}
	bindings.saved = append(bindings.saved, domain.RepositoryBinding{
		ID: "bnd_missing", AccountID: account.ID,
		Repository: domain.RepositoryRef{Path: "/repo-gone"}, Strategy: domain.StrategyRepoLocal, Revision: 1,
	})
	git.missing["/repo-gone"] = true
	git.values["user.name"] = "Drifted Name"

	result, err := NewReconciler(service.deps).ReconcileAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("ReconcileAccount() error = %v", err)
	}
	if result.Changed == 0 {
		t.Fatalf("result = %+v, want changes for the healthy binding", result)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures = %v, want the missing repository reported", result.Failures)
	}
	if git.values["user.name"] != "Luna" {
		t.Fatalf("drift not repaired: %q", git.values["user.name"])
	}
	if bound.ID == "" {
		t.Fatal("bind lost")
	}
}
