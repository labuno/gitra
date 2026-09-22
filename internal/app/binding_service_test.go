package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"
)

// ---- fakes ----

type fakeGit struct {
	values  map[string]string
	global  map[string]string
	remote  string
	missing map[string]bool
}

func newFakeGit(remote string) *fakeGit {
	return &fakeGit{
		values:  map[string]string{"user.name": "Old Name", "user.email": "old@example.com"},
		global:  map[string]string{},
		remote:  remote,
		missing: map[string]bool{},
	}
}

func (f *fakeGit) DiscoverRepository(_ context.Context, path string) (domain.Repository, error) {
	if f.missing[path] {
		return domain.Repository{}, domain.ErrNotGitRepository
	}
	repo := domain.Repository{RootPath: path, GitDir: path + "/.git"}
	if f.remote != "" {
		repo.Remotes = []domain.Remote{{Name: "origin", URL: f.remote}}
	}
	return repo, nil
}
func (f *fakeGit) GetGlobalConfig(_ context.Context, key string) (string, bool, error) {
	value, ok := f.global[key]
	return value, ok, nil
}

func (f *fakeGit) GetLocalConfig(_ context.Context, _ domain.Repository, key string) (string, bool, error) {
	value, ok := f.values[key]
	return value, ok, nil
}
func (f *fakeGit) SetLocalConfig(_ context.Context, _ domain.Repository, key, value string) error {
	f.values[key] = value
	return nil
}
func (f *fakeGit) UnsetLocalConfig(_ context.Context, _ domain.Repository, key string) error {
	delete(f.values, key)
	return nil
}
func (f *fakeGit) Remotes(context.Context, domain.Repository) ([]domain.Remote, error) {
	return nil, nil
}

type fakeAccounts struct {
	account domain.Account
	missing bool
}

func (f *fakeAccounts) Save(context.Context, domain.Account) error { return nil }
func (f *fakeAccounts) Get(context.Context, domain.AccountID) (domain.Account, error) {
	if f.missing {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	return f.account, nil
}
func (f *fakeAccounts) List(context.Context) ([]domain.Account, error) {
	return []domain.Account{f.account}, nil
}
func (f *fakeAccounts) Delete(context.Context, domain.AccountID) error { return nil }

type fakeBindings struct {
	saved    []domain.RepositoryBinding
	failSave bool
}

func (f *fakeBindings) Save(_ context.Context, b domain.RepositoryBinding) error {
	if f.failSave {
		return errors.New("injected central store failure")
	}
	for i, existing := range f.saved {
		if existing.ID == b.ID {
			f.saved[i] = b
			return nil
		}
	}
	f.saved = append(f.saved, b)
	return nil
}
func (f *fakeBindings) Get(_ context.Context, id domain.BindingID) (domain.RepositoryBinding, error) {
	for _, b := range f.saved {
		if b.ID == id {
			return b, nil
		}
	}
	return domain.RepositoryBinding{}, domain.ErrBindingNotFound
}
func (f *fakeBindings) FindByRepository(_ context.Context, path string) (domain.RepositoryBinding, bool, error) {
	for _, b := range f.saved {
		if b.Repository.Path == path {
			return b, true, nil
		}
	}
	return domain.RepositoryBinding{}, false, nil
}
func (f *fakeBindings) ListByAccount(_ context.Context, id domain.AccountID) ([]domain.RepositoryBinding, error) {
	var out []domain.RepositoryBinding
	for _, b := range f.saved {
		if b.AccountID == id {
			out = append(out, b)
		}
	}
	return out, nil
}
func (f *fakeBindings) Delete(_ context.Context, id domain.BindingID) error {
	kept := f.saved[:0]
	for _, b := range f.saved {
		if b.ID != id {
			kept = append(kept, b)
		}
	}
	f.saved = kept
	return nil
}

type fakeSnapshots struct{ byGitDir map[string]ports.Snapshot }

func newFakeSnapshots() *fakeSnapshots { return &fakeSnapshots{byGitDir: map[string]ports.Snapshot{}} }
func (f *fakeSnapshots) Load(gitDir string) (ports.Snapshot, bool, error) {
	s, ok := f.byGitDir[gitDir]
	return s, ok, nil
}
func (f *fakeSnapshots) Save(gitDir string, s ports.Snapshot) error {
	f.byGitDir[gitDir] = s
	return nil
}
func (f *fakeSnapshots) Delete(gitDir string) error {
	delete(f.byGitDir, gitDir)
	return nil
}

type noopLocker struct{}

func (noopLocker) WithWriteLock(_ context.Context, fn func() error) error { return fn() }

// countingLocker records how many write transactions were started.
type countingLocker struct{ acquired int }

func (c *countingLocker) WithWriteLock(_ context.Context, fn func() error) error {
	c.acquired++
	return fn()
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Unix(0, 0) }

// ---- helpers ----

func testAccount(t *testing.T) domain.Account {
	t.Helper()
	keyPath := filepath.Join(t.TempDir(), "id_ed25519_luna")
	if err := os.WriteFile(keyPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	return domain.Account{
		ID:    "acc_001",
		Alias: "luna",
		Provider: domain.ProviderRef{
			Type:     domain.ProviderGitHub,
			Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": keyPath}},
		Revision:  1,
	}
}

func newService(t *testing.T, remote string) (*BindingService, *fakeGit, *fakeBindings, *fakeSnapshots, domain.Account) {
	t.Helper()
	account := testAccount(t)
	git := newFakeGit(remote)
	bindings := &fakeBindings{}
	snapshots := newFakeSnapshots()

	authRegistry := auth.NewRegistry()
	if err := authRegistry.Register(sshkey.New()); err != nil {
		t.Fatal(err)
	}
	routingRegistry := routing.NewRegistry()
	if err := routingRegistry.Register(repolocal.New()); err != nil {
		t.Fatal(err)
	}
	service := NewBindingService(Deps{
		Accounts:  &fakeAccounts{account: account},
		Bindings:  bindings,
		Git:       git,
		Snapshots: snapshots,
		Auth:      authRegistry,
		Routing:   routingRegistry,
		Locker:    noopLocker{},
		Clock:     fixedClock{},
	})
	return service, git, bindings, snapshots, account
}

// ---- tests ----

func TestBindWritesManagedStateAndBinding(t *testing.T) {
	service, git, bindings, snapshots, account := newService(t, "git@github.com:lunafoundry/luna-site.git")
	ctx := context.Background()

	binding, err := service.Bind(ctx, BindRequest{AccountID: account.ID, Path: "/repo"})
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if binding.Strategy != domain.StrategyRepoLocal || binding.Repository.Path != "/repo" {
		t.Fatalf("binding = %+v", binding)
	}
	want := map[string]string{
		"user.name":       "Luna",
		"user.email":      "luna@example.com",
		"gitra.bindingId": string(binding.ID),
		"gitra.accountId": string(account.ID),
		"gitra.strategy":  "repo-local",
		"gitra.version":   "1",
	}
	for key, value := range want {
		if git.values[key] != value {
			t.Fatalf("%s = %q, want %q", key, git.values[key], value)
		}
	}
	if len(bindings.saved) != 1 {
		t.Fatalf("saved bindings = %+v", bindings.saved)
	}
	snapshot, ok, _ := snapshots.Load("/repo/.git")
	if !ok || snapshot.BindingID != binding.ID || snapshot.Previous["user.email"].Value != "old@example.com" {
		t.Fatalf("snapshot = %+v (ok=%v)", snapshot, ok)
	}
}

func TestBindRejectsAlreadyBound(t *testing.T) {
	service, _, bindings, _, account := newService(t, "")
	bindings.saved = append(bindings.saved, domain.RepositoryBinding{
		ID: "bnd_existing", AccountID: account.ID,
		Repository: domain.RepositoryRef{Path: "/repo"}, Strategy: domain.StrategyRepoLocal, Revision: 1,
	})
	if _, err := service.Bind(context.Background(), BindRequest{AccountID: account.ID, Path: "/repo"}); !errors.Is(err, domain.ErrAlreadyBound) {
		t.Fatalf("error = %v, want ErrAlreadyBound", err)
	}
}

func TestBindRejectsUnsupportedRemoteAndProviderMismatch(t *testing.T) {
	service, _, _, _, account := newService(t, "https://github.com/lunafoundry/luna-site.git")
	if _, err := service.Bind(context.Background(), BindRequest{AccountID: account.ID, Path: "/repo"}); !errors.Is(err, domain.ErrUnsupportedRemote) {
		t.Fatalf("https error = %v, want ErrUnsupportedRemote", err)
	}

	service, _, _, _, account = newService(t, "git@git.example.com:luna/repo.git")
	if _, err := service.Bind(context.Background(), BindRequest{AccountID: account.ID, Path: "/repo"}); !errors.Is(err, domain.ErrProviderMismatch) {
		t.Fatalf("mismatch error = %v, want ErrProviderMismatch", err)
	}
}

func TestBindRollsBackOnStoreFailure(t *testing.T) {
	service, git, bindings, snapshots, account := newService(t, "git@github.com:lunafoundry/luna-site.git")
	bindings.failSave = true

	if _, err := service.Bind(context.Background(), BindRequest{AccountID: account.ID, Path: "/repo"}); err == nil {
		t.Fatal("Bind() must fail when the central store fails")
	}
	if git.values["user.email"] != "old@example.com" || git.values["user.name"] != "Old Name" {
		t.Fatalf("identity not restored: %+v", git.values)
	}
	for _, key := range []string{"core.sshCommand", "gitra.bindingId", "gitra.accountId", "gitra.strategy", "gitra.version"} {
		if _, present := git.values[key]; present {
			t.Fatalf("managed key %s left behind after rollback: %+v", key, git.values)
		}
	}
	if _, ok, _ := snapshots.Load("/repo/.git"); ok {
		t.Fatal("snapshot must be removed after rollback")
	}
}

func TestUnbindRestoresPreviousState(t *testing.T) {
	service, git, bindings, snapshots, account := newService(t, "git@github.com:lunafoundry/luna-site.git")
	ctx := context.Background()
	if _, err := service.Bind(ctx, BindRequest{AccountID: account.ID, Path: "/repo"}); err != nil {
		t.Fatal(err)
	}
	if err := service.Unbind(ctx, UnbindRequest{Path: "/repo"}); err != nil {
		t.Fatalf("Unbind() error = %v", err)
	}
	if git.values["user.email"] != "old@example.com" || git.values["user.name"] != "Old Name" {
		t.Fatalf("values after unbind = %+v", git.values)
	}
	if _, present := git.values["core.sshCommand"]; present {
		t.Fatalf("core.sshCommand must be unset: %+v", git.values)
	}
	if len(bindings.saved) != 0 {
		t.Fatalf("binding not removed: %+v", bindings.saved)
	}
	if _, ok, _ := snapshots.Load("/repo/.git"); ok {
		t.Fatal("snapshot must be removed after unbind")
	}
}

func TestStatusUnboundAndBound(t *testing.T) {
	service, _, _, _, account := newService(t, "git@github.com:lunafoundry/luna-site.git")
	ctx := context.Background()

	status, err := service.Status(ctx, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if status.Bound || status.Health != "" {
		t.Fatalf("unbound status = %+v", status)
	}

	if _, err := service.Bind(ctx, BindRequest{AccountID: account.ID, Path: "/repo"}); err != nil {
		t.Fatal(err)
	}
	status, err = service.Status(ctx, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Bound || status.Health != domain.HealthOK {
		t.Fatalf("bound status = %+v", status)
	}
}
