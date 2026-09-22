package app

import (
	"context"
	"errors"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/httptoken"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"
)

type fakeSecrets struct{ values map[string]string }

func (f *fakeSecrets) Set(_ context.Context, ref, secret string) error {
	f.values[ref] = secret
	return nil
}
func (f *fakeSecrets) Get(_ context.Context, ref string) (string, error) {
	if value, ok := f.values[ref]; ok {
		return value, nil
	}
	return "", domain.ErrAuthInvalid
}
func (f *fakeSecrets) Delete(_ context.Context, ref string) error {
	delete(f.values, ref)
	return nil
}

type fakeHelperResolver struct {
	spec string
	ok   bool
}

func (f fakeHelperResolver) Helper(context.Context) (string, bool, error) { return f.spec, f.ok, nil }

func httpsAccount() domain.Account {
	return domain.Account{
		ID:    "acc_https",
		Alias: "luna",
		Provider: domain.ProviderRef{
			Type:     domain.ProviderGitHub,
			Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategyHTTPSToken, Config: map[string]string{}},
		Revision:  1,
	}
}

type httpsEnv struct {
	service  *BindingService
	git      *fakeGit
	bindings *fakeBindings
	secrets  *fakeSecrets
	deps     Deps
	account  domain.Account
}

func newHTTPSEnv(t *testing.T, remote string) *httpsEnv {
	t.Helper()
	account := httpsAccount()
	git := newFakeGit(remote)
	secrets := &fakeSecrets{values: map[string]string{"github.com/lunafoundry": "gho_token"}}

	authRegistry := auth.NewRegistry()
	if err := authRegistry.Register(sshkey.New()); err != nil {
		t.Fatal(err)
	}
	if err := authRegistry.Register(httptoken.New(secrets)); err != nil {
		t.Fatal(err)
	}
	routingRegistry := routing.NewRegistry()
	if err := routingRegistry.Register(repolocal.New()); err != nil {
		t.Fatal(err)
	}
	deps := Deps{
		Accounts:  &fakeAccounts{account: account},
		Bindings:  &fakeBindings{},
		Git:       git,
		Snapshots: newFakeSnapshots(),
		Auth:      authRegistry,
		Routing:   routingRegistry,
		Locker:    noopLocker{},
		Clock:     fixedClock{},
		Secrets:   secrets, CredentialHelpers: fakeHelperResolver{spec: "store --file=/tmp/creds", ok: true},
	}
	return &httpsEnv{
		service:  NewBindingService(deps),
		git:      git,
		bindings: deps.Bindings.(*fakeBindings),
		secrets:  secrets,
		deps:     deps,
		account:  account,
	}
}

func TestBindHTTPSAccountProjectsCredentialScope(t *testing.T) {
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	ctx := context.Background()

	binding, err := env.service.Bind(ctx, BindRequest{AccountID: env.account.ID, Path: "/repo"})
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if env.git.values["credential.https://github.com.username"] != "lunafoundry" {
		t.Fatalf("credential scope missing: %+v", env.git.values)
	}
	if env.git.values["credential.helper"] != "store --file=/tmp/creds" {
		t.Fatalf("helper entry missing: %+v", env.git.values)
	}
	if env.git.values["user.email"] != "luna@example.com" {
		t.Fatalf("identity missing: %+v", env.git.values)
	}
	if _, leaked := env.git.values["core.sshCommand"]; leaked {
		t.Fatalf("https account must not write core.sshCommand: %+v", env.git.values)
	}

	if err := env.service.Unbind(ctx, UnbindRequest{Path: "/repo"}); err != nil {
		t.Fatalf("Unbind() error = %v", err)
	}
	for _, key := range []string{"credential.https://github.com.username", "credential.helper", "gitra.bindingId"} {
		if _, present := env.git.values[key]; present {
			t.Fatalf("%s must be removed after unbind: %+v", key, env.git.values)
		}
	}
	if env.git.values["user.email"] != "old@example.com" {
		t.Fatalf("identity not restored: %+v", env.git.values)
	}
	if len(env.bindings.saved) != 0 {
		t.Fatalf("binding not removed: %+v", env.bindings.saved)
	}
	if binding.ID == "" {
		t.Fatal("binding id missing")
	}
}

func TestBindRejectsTransportMismatch(t *testing.T) {
	ctx := context.Background()

	httpsEnv := newHTTPSEnv(t, "git@github.com:lunafoundry/luna-site.git")
	if _, err := httpsEnv.service.Bind(ctx, BindRequest{AccountID: httpsEnv.account.ID, Path: "/repo"}); !errors.Is(err, domain.ErrUnsupportedRemote) {
		t.Fatalf("https account + ssh remote error = %v, want ErrUnsupportedRemote", err)
	}

	sshEnv := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	sshAccount := httpsAccount()
	sshAccount.Transport.Strategy = domain.StrategySSHKey
	sshAccount.Transport.Config = map[string]string{"private_key": "/tmp/does-not-matter"}
	sshEnv.deps.Accounts = &fakeAccounts{account: sshAccount}
	sshEnv.deps.Auth = mustAuthRegistry(t)
	service := NewBindingService(sshEnv.deps)
	if _, err := service.Bind(ctx, BindRequest{AccountID: sshAccount.ID, Path: "/repo"}); !errors.Is(err, domain.ErrUnsupportedRemote) {
		t.Fatalf("ssh account + https remote error = %v, want ErrUnsupportedRemote", err)
	}
}

func TestStatusNeedsLoginWhenCredentialMissing(t *testing.T) {
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	ctx := context.Background()
	if _, err := env.service.Bind(ctx, BindRequest{AccountID: env.account.ID, Path: "/repo"}); err != nil {
		t.Fatal(err)
	}
	delete(env.secrets.values, "github.com/lunafoundry")

	status, err := env.service.Status(ctx, "/repo")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Bound || !status.NeedsLogin || status.Health != domain.HealthBroken {
		t.Fatalf("status = %+v, want needs_login/broken", status)
	}

	result, err := NewReconciler(env.deps).ReconcileBinding(ctx, status.Binding.ID)
	if err != nil {
		t.Fatalf("ReconcileBinding() error = %v", err)
	}
	if result.Health != domain.HealthBroken || len(result.Failures) == 0 {
		t.Fatalf("reconcile result = %+v", result)
	}
}

func mustAuthRegistry(t *testing.T) *auth.Registry {
	t.Helper()
	registry := auth.NewRegistry()
	if err := registry.Register(sshkey.New()); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(httptoken.New(&fakeSecrets{values: map[string]string{}})); err != nil {
		t.Fatal(err)
	}
	return registry
}
