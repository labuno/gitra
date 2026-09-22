package app

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"
)

// memAccounts is an in-memory ports.AccountStore with alias uniqueness.
type memAccounts struct {
	byID map[domain.AccountID]domain.Account
}

func newMemAccounts() *memAccounts { return &memAccounts{byID: map[domain.AccountID]domain.Account{}} }

func (m *memAccounts) Save(_ context.Context, account domain.Account) error {
	for _, other := range m.byID {
		if other.Alias == account.Alias && other.ID != account.ID {
			return domain.ErrAccountExists
		}
	}
	m.byID[account.ID] = account
	return nil
}
func (m *memAccounts) Get(_ context.Context, id domain.AccountID) (domain.Account, error) {
	account, ok := m.byID[id]
	if !ok {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	return account, nil
}
func (m *memAccounts) List(context.Context) ([]domain.Account, error) {
	var out []domain.Account
	for _, account := range m.byID {
		out = append(out, account)
	}
	return out, nil
}
func (m *memAccounts) Delete(_ context.Context, id domain.AccountID) error {
	if _, ok := m.byID[id]; !ok {
		return domain.ErrAccountNotFound
	}
	delete(m.byID, id)
	return nil
}

type accountEnv struct {
	service    *AccountService
	accounts   *memAccounts
	git        *fakeGit
	bindings   *fakeBindings
	account    domain.Account
	reconciler *Reconciler
}

func newAccountEnv(t *testing.T) *accountEnv {
	t.Helper()
	account := testAccount(t)
	store := newMemAccounts()
	if err := store.Save(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	git := newFakeGit("git@github.com:lunafoundry/luna-site.git")
	authRegistry := auth.NewRegistry()
	if err := authRegistry.Register(sshkey.New()); err != nil {
		t.Fatal(err)
	}
	routingRegistry := routing.NewRegistry()
	if err := routingRegistry.Register(repolocal.New()); err != nil {
		t.Fatal(err)
	}
	deps := Deps{
		Accounts:  store,
		Bindings:  &fakeBindings{},
		Git:       git,
		Snapshots: newFakeSnapshots(),
		Auth:      authRegistry,
		Routing:   routingRegistry,
		Locker:    noopLocker{},
		Clock:     fixedClock{},
	}
	accountService := NewAccountService(deps, NewBindingService(deps))
	return &accountEnv{
		service:    accountService,
		accounts:   store,
		git:        git,
		bindings:   deps.Bindings.(*fakeBindings),
		account:    account,
		reconciler: NewReconciler(deps),
	}
}

func TestAccountCreate(t *testing.T) {
	env := newAccountEnv(t)
	ctx := context.Background()

	created, err := env.service.Create(ctx, CreateAccountRequest{
		Alias:     "new-alias",
		Provider:  env.account.Provider,
		Identity:  env.account.Identity,
		Transport: env.account.Transport,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Revision != 1 || len(string(created.ID)) < 5 {
		t.Fatalf("created = %+v", created)
	}
	if _, err := env.service.Get(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := env.service.Create(ctx, CreateAccountRequest{
		Alias: "Bad Alias", Provider: env.account.Provider, Identity: env.account.Identity, Transport: env.account.Transport,
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid alias error = %v, want ErrInvalid", err)
	}
	if _, err := env.service.Create(ctx, CreateAccountRequest{
		Alias: env.account.Alias, Provider: env.account.Provider, Identity: env.account.Identity, Transport: env.account.Transport,
	}); !errors.Is(err, domain.ErrAccountExists) {
		t.Fatalf("duplicate alias error = %v, want ErrAccountExists", err)
	}
}

func TestAccountUpdatePropagatesToBoundRepositories(t *testing.T) {
	env := newAccountEnv(t)
	ctx := context.Background()
	bindingService := NewBindingService(env.service.deps)
	if _, err := bindingService.Bind(ctx, BindRequest{AccountID: env.account.ID, Path: "/repo"}); err != nil {
		t.Fatal(err)
	}

	next := env.account
	next.Identity.Email = "new@example.com"
	updated, err := env.service.Update(ctx, UpdateAccountRequest{Account: next})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Revision != env.account.Revision+1 {
		t.Fatalf("revision = %d, want %d", updated.Revision, env.account.Revision+1)
	}
	if env.git.values["user.email"] != "new@example.com" {
		t.Fatalf("reconcile did not propagate: user.email = %q", env.git.values["user.email"])
	}
}

func TestAccountDeleteGuards(t *testing.T) {
	env := newAccountEnv(t)
	ctx := context.Background()
	bindingService := NewBindingService(env.service.deps)
	if _, err := bindingService.Bind(ctx, BindRequest{AccountID: env.account.ID, Path: "/repo"}); err != nil {
		t.Fatal(err)
	}

	if err := env.service.Delete(ctx, DeleteAccountRequest{ID: env.account.ID}); !errors.Is(err, domain.ErrAccountInUse) {
		t.Fatalf("delete with binding error = %v, want ErrAccountInUse", err)
	}
	if err := env.service.Delete(ctx, DeleteAccountRequest{ID: env.account.ID, UnbindAll: true}); err != nil {
		t.Fatalf("UnbindAll delete error = %v", err)
	}
	if _, err := env.service.Get(ctx, env.account.ID); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Fatalf("account still present: %v", err)
	}
	if len(env.bindings.saved) != 0 {
		t.Fatalf("bindings not removed: %+v", env.bindings.saved)
	}
}

func TestAccountGetByAliasAndID(t *testing.T) {
	env := newAccountEnv(t)
	ctx := context.Background()
	if _, err := env.service.GetByAlias(ctx, "luna"); err != nil {
		t.Fatalf("by alias: %v", err)
	}
	if _, err := env.service.GetByAlias(ctx, string(env.account.ID)); err != nil {
		t.Fatalf("by id: %v", err)
	}
	if _, err := env.service.GetByAlias(ctx, "missing"); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Fatalf("missing alias error = %v", err)
	}
}
