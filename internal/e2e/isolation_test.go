// Package e2e holds the long-lived end-to-end regression tests (baseline §53.7).
// They exercise real git repositories, the real adapters and an isolated
// config directory (GITRA_CONFIG_DIR), never the developer's own settings.
package e2e

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/gitcli"
	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/adapters/storage"
	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"
)

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func gitConfig(t *testing.T, dir, key string) (string, bool) {
	t.Helper()
	cmd := exec.Command("git", "config", "--local", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func gitGlobalOrEmpty(t *testing.T, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--global", "--get", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func makeRepo(t *testing.T, name, remoteURL string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "config", "user.name", "Old Name")
	gitRun(t, root, "config", "user.email", "old@example.com")
	gitRun(t, root, "remote", "add", "origin", remoteURL)
	return root
}

func makeAccount(t *testing.T, id, alias, username, email, keyName string) domain.Account {
	t.Helper()
	keyDir := t.TempDir()
	keyPath := filepath.Join(keyDir, keyName)
	if err := os.WriteFile(keyPath, []byte("placeholder private key"), 0o600); err != nil {
		t.Fatal(err)
	}
	return domain.Account{
		ID:    domain.AccountID(id),
		Alias: alias,
		Provider: domain.ProviderRef{
			Type:     domain.ProviderGitHub,
			Username: username,
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: alias, Email: email},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": keyPath}},
		Revision:  1,
	}
}

type env struct {
	deps     app.Deps
	service  *app.BindingService
	accounts []domain.Account
	repos    []string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	configDir := t.TempDir()
	t.Setenv("GITRA_CONFIG_DIR", configDir)

	authRegistry := auth.NewRegistry()
	if err := authRegistry.Register(sshkey.New()); err != nil {
		t.Fatal(err)
	}
	routingRegistry := routing.NewRegistry()
	if err := routingRegistry.Register(repolocal.New()); err != nil {
		t.Fatal(err)
	}

	deps := app.Deps{
		Accounts:  storage.NewAccountStore(configDir),
		Bindings:  storage.NewBindingStore(configDir),
		Git:       gitcli.New(runner.New()),
		Snapshots: storage.NewSnapshotStore(),
		Auth:      authRegistry,
		Routing:   routingRegistry,
		Locker:    storage.NewFileLocker(filepath.Join(configDir, ".lock")),
	}
	accounts := []domain.Account{
		makeAccount(t, "acc_001", "luna", "lunafoundry", "luna@example.com", "id_ed25519_luna"),
		makeAccount(t, "acc_002", "work", "luna-work", "work@example.com", "id_ed25519_work"),
	}
	for _, account := range accounts {
		if err := deps.Accounts.Save(context.Background(), account); err != nil {
			t.Fatal(err)
		}
	}
	return &env{
		deps:     deps,
		service:  app.NewBindingService(deps),
		accounts: accounts,
		repos: []string{
			makeRepo(t, "luna-site", "git@github.com:lunafoundry/luna-site.git"),
			makeRepo(t, "company-api", "git@github.com:lunacorp/company-api.git"),
		},
	}
}

func TestDualAccountDualRepositoryIsolation(t *testing.T) {
	env := newEnv(t)
	ctx := context.Background()

	globalBefore := gitGlobalOrEmpty(t, "user.email") // may be empty; must not change

	for i, repo := range env.repos {
		if _, err := env.service.Bind(ctx, app.BindRequest{AccountID: env.accounts[i].ID, Path: repo}); err != nil {
			t.Fatalf("Bind(%s) error = %v", repo, err)
		}
	}

	emailA, _ := gitConfig(t, env.repos[0], "user.email")
	emailB, _ := gitConfig(t, env.repos[1], "user.email")
	if emailA != "luna@example.com" || emailB != "work@example.com" {
		t.Fatalf("identity leak: repo A %q, repo B %q", emailA, emailB)
	}
	sshA, _ := gitConfig(t, env.repos[0], "core.sshCommand")
	sshB, _ := gitConfig(t, env.repos[1], "core.sshCommand")
	if sshA == "" || sshB == "" || sshA == sshB {
		t.Fatalf("ssh key leak: A %q, B %q", sshA, sshB)
	}
	if !strings.Contains(sshA, env.accounts[0].Transport.Config["private_key"]) ||
		!strings.Contains(sshB, env.accounts[1].Transport.Config["private_key"]) {
		t.Fatalf("sshCommand does not reference the account key: A %q, B %q", sshA, sshB)
	}

	// remotes must be untouched
	if got := gitRun(t, env.repos[0], "remote", "get-url", "origin"); got != "git@github.com:lunafoundry/luna-site.git" {
		t.Fatalf("remote rewritten: %q", got)
	}

	// metadata identifies the binding
	bindingA, ok := gitConfig(t, env.repos[0], "gitra.bindingId")
	status, err := env.service.Status(ctx, env.repos[0])
	if err != nil || !status.Bound || status.Health != domain.HealthOK {
		t.Fatalf("status = %+v, err = %v", status, err)
	}
	if !ok || bindingA != string(status.Binding.ID) || status.Account.ID != env.accounts[0].ID {
		t.Fatalf("metadata mismatch: %q vs %+v", bindingA, status)
	}

	// unbound third repository carries no managed keys
	unbound := makeRepo(t, "unbound", "git@github.com:lunafoundry/unbound.git")
	if _, present := gitConfig(t, unbound, "gitra.bindingId"); present {
		t.Fatal("unbound repository must not carry gitra metadata")
	}

	// global git config untouched
	if after := gitGlobalOrEmpty(t, "user.email"); after != globalBefore {
		t.Fatalf("global user.email changed: %q -> %q", globalBefore, after)
	}
}

func TestUnbindRestoresPreBindState(t *testing.T) {
	env := newEnv(t)
	ctx := context.Background()
	repo := env.repos[0]
	if _, err := env.service.Bind(ctx, app.BindRequest{AccountID: env.accounts[0].ID, Path: repo}); err != nil {
		t.Fatal(err)
	}
	if err := env.service.Unbind(ctx, app.UnbindRequest{Path: repo}); err != nil {
		t.Fatalf("Unbind() error = %v", err)
	}
	if name, _ := gitConfig(t, repo, "user.name"); name != "Old Name" {
		t.Fatalf("user.name = %q, want Old Name", name)
	}
	if email, _ := gitConfig(t, repo, "user.email"); email != "old@example.com" {
		t.Fatalf("user.email = %q, want old@example.com", email)
	}
	for _, key := range []string{"core.sshCommand", "gitra.bindingId", "gitra.accountId", "gitra.strategy", "gitra.version"} {
		if _, present := gitConfig(t, repo, key); present {
			t.Fatalf("%s must be removed after unbind", key)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "gitra", "state.json")); !os.IsNotExist(err) {
		t.Fatalf("snapshot must be deleted, stat err = %v", err)
	}
}

type failingBindings struct{ ports.BindingStore }

func (failingBindings) Save(context.Context, domain.RepositoryBinding) error {
	return errors.New("injected central store failure")
}

func TestBindRollsBackE2E(t *testing.T) {
	env := newEnv(t)
	ctx := context.Background()
	deps := env.deps
	deps.Bindings = failingBindings{env.deps.Bindings}
	service := app.NewBindingService(deps)
	repo := env.repos[0]

	if _, err := service.Bind(ctx, app.BindRequest{AccountID: env.accounts[0].ID, Path: repo}); err == nil {
		t.Fatal("Bind() must fail with the injected store failure")
	}
	if name, _ := gitConfig(t, repo, "user.name"); name != "Old Name" {
		t.Fatalf("user.name = %q after rollback, want Old Name", name)
	}
	if email, _ := gitConfig(t, repo, "user.email"); email != "old@example.com" {
		t.Fatalf("user.email = %q after rollback", email)
	}
	for _, key := range []string{"core.sshCommand", "gitra.bindingId", "gitra.accountId"} {
		if _, present := gitConfig(t, repo, key); present {
			t.Fatalf("%s left behind after rollback", key)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", "gitra", "state.json")); !os.IsNotExist(err) {
		t.Fatalf("snapshot must be removed after rollback, stat err = %v", err)
	}
}
