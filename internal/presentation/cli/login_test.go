package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/gitcli"
	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/adapters/secretstore"
	"github.com/zhanhd/gitra/internal/adapters/storage"
	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/httptoken"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"
)

type stubTokens struct {
	token ports.LoginToken
	err   error
}

func (s stubTokens) Acquire(context.Context, domain.ProviderType, string, bool, bool) (ports.LoginToken, error) {
	return s.token, s.err
}

type stubProfiles struct{ profile ports.ProviderProfile }

func (s stubProfiles) Profile(context.Context, domain.ProviderType, string, string) (ports.ProviderProfile, error) {
	return s.profile, nil
}

type loginEnv struct {
	cli         *App
	application *bootstrap.App
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
	deps        app.Deps
}

func newLoginEnv(t *testing.T, tokens stubTokens, profiles stubProfiles) *loginEnv {
	t.Helper()
	configDir := t.TempDir()
	t.Setenv("GITRA_CONFIG_DIR", configDir)
	gitRunner := runner.New()
	secrets := secretstore.NewGitCredentialStore(gitRunner, "store --file="+filepath.Join(configDir, "credentials"))

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
	deps := app.Deps{
		Accounts:          storage.NewAccountStore(configDir),
		Bindings:          storage.NewBindingStore(configDir),
		Git:               gitcli.New(gitRunner),
		Snapshots:         storage.NewSnapshotStore(),
		Auth:              authRegistry,
		Routing:           routingRegistry,
		Locker:            storage.NewFileLocker(filepath.Join(configDir, ".lock")),
		Secrets:           secrets,
		CredentialHelpers: secretstore.NewHelperResolver(gitRunner, configDir),
	}
	application := bootstrap.NewFromDeps(deps, tokens, profiles)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return &loginEnv{cli: New(application, stdout, stderr), application: application, stdout: stdout, stderr: stderr, deps: deps}
}

func (e *loginEnv) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	e.stdout.Reset()
	e.stderr.Reset()
	code := e.cli.Execute(context.Background(), args)
	return code, e.stdout.String(), e.stderr.String()
}

func gitConfigOrEmpty(t *testing.T, dir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--local", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func githubProfile() ports.ProviderProfile {
	return ports.ProviderProfile{
		Provider: domain.ProviderGitHub, Host: "github.com",
		Username: "lunafoundry", Name: "Luna", Email: "luna@example.com",
	}
}

func TestLoginCreatesHTTPSAccountWithoutLeakingToken(t *testing.T) {
	env := newLoginEnv(t, stubTokens{token: ports.LoginToken{Value: "gho_supersecret", Source: "stdin"}}, stubProfiles{profile: githubProfile()})

	code, stdout, stderr := env.run(t, "login", "--provider", "github", "--token-stdin")
	if code != 0 {
		t.Fatalf("login exit=%d stderr=%s", code, stderr)
	}
	if strings.Contains(stdout+stderr, "gho_supersecret") {
		t.Fatal("token leaked into command output")
	}
	secret, err := env.deps.Secrets.Get(context.Background(), "github.com/lunafoundry")
	if err != nil || secret != "gho_supersecret" {
		t.Fatalf("stored secret = (%q, %v)", secret, err)
	}

	code, stdout, stderr = env.run(t, "account", "list", "--json")
	if code != 0 {
		t.Fatalf("list exit=%d stderr=%s", code, stderr)
	}
	var list struct {
		Accounts []struct {
			ID    string `json:"id"`
			Alias string `json:"alias"`
			Auth  struct {
				Strategy string `json:"strategy"`
				State    string `json:"state"`
			} `json:"auth"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Accounts) != 1 || list.Accounts[0].Alias != "lunafoundry" ||
		list.Accounts[0].Auth.Strategy != "https-token" || list.Accounts[0].Auth.State != "configured" {
		t.Fatalf("accounts = %+v", list.Accounts)
	}
}

func TestLoginSecondRunUpdatesExistingAccount(t *testing.T) {
	env := newLoginEnv(t, stubTokens{token: ports.LoginToken{Value: "tok", Source: "env"}}, stubProfiles{profile: githubProfile()})
	if code, _, stderr := env.run(t, "login", "--provider", "github"); code != 0 {
		t.Fatalf("first login exit=%d stderr=%s", code, stderr)
	}
	accounts, err := env.application.Accounts.List(context.Background())
	if err != nil || len(accounts) != 1 {
		t.Fatalf("accounts = %v, %v", accounts, err)
	}
	firstID := accounts[0].ID

	code, stdout, stderr := env.run(t, "login", "--provider", "github", "--alias", "renamed")
	if code != 0 {
		t.Fatalf("second login exit=%d stderr=%s", code, stderr)
	}
	accounts, err = env.application.Accounts.List(context.Background())
	if err != nil || len(accounts) != 1 {
		t.Fatalf("second login created a duplicate: %v, %v", accounts, err)
	}
	if accounts[0].ID != firstID || accounts[0].Alias != "renamed" || accounts[0].Revision != 2 {
		t.Fatalf("account after second login = %+v", accounts[0])
	}
	if !strings.Contains(stdout, "renamed") {
		t.Fatalf("stdout = %s", stdout)
	}
}

func TestLoginWithoutCredentialExplainsOptions(t *testing.T) {
	env := newLoginEnv(t, stubTokens{err: fmt.Errorf(
		"%w: no credential available for github.\nProvide one with:\n  gitra login --provider github --token-stdin   (paste a personal access token)\nor export GITRA_TOKEN.",
		domain.ErrAuthInvalid)}, stubProfiles{profile: githubProfile()})
	code, _, stderr := env.run(t, "login", "--provider", "github")
	if code != 5 || !strings.Contains(stderr, "--token-stdin") {
		t.Fatalf("exit=%d stderr=%s, want 5 with guidance", code, stderr)
	}
}

func TestLogoutClearsCredentialScopeAndReportsNeedsLogin(t *testing.T) {
	env := newLoginEnv(t, stubTokens{token: ports.LoginToken{Value: "tok", Source: "stdin"}}, stubProfiles{profile: githubProfile()})
	ctx := context.Background()
	if code, _, stderr := env.run(t, "login", "--provider", "github", "--token-stdin"); code != 0 {
		t.Fatalf("login exit=%d stderr=%s", code, stderr)
	}

	repo := filepath.Join(t.TempDir(), "site")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRunT(t, repo, "init", "-q")
	gitRunT(t, repo, "config", "user.name", "Old Name")
	gitRunT(t, repo, "config", "user.email", "old@example.com")
	gitRunT(t, repo, "remote", "add", "origin", "https://github.com/lunafoundry/site.git")

	accounts, _ := env.application.Accounts.List(ctx)
	if _, err := env.application.Bindings.Bind(ctx, app.BindRequest{AccountID: accounts[0].ID, Path: repo}); err != nil {
		t.Fatalf("bind error = %v", err)
	}

	if code, _, stderr := env.run(t, "logout", "lunafoundry"); code != 0 {
		t.Fatalf("logout exit=%d stderr=%s", code, stderr)
	}
	for _, key := range []string{"credential.https://github.com.username", "credential.helper"} {
		if got := gitConfigOrEmpty(t, repo, key); got != "" {
			t.Fatalf("%s still present after logout: %q", key, got)
		}
	}
	if _, err := env.deps.Secrets.Get(ctx, "github.com/lunafoundry"); err == nil {
		t.Fatal("credential still stored after logout")
	}

	code, stdout, _ := env.run(t, "status", repo, "--json")
	if code != 6 {
		t.Fatalf("status exit=%d, want 6 after logout", code)
	}
	var status struct {
		Auth *struct {
			State string `json:"state"`
		} `json:"auth"`
		Binding *struct {
			Health string `json:"health"`
		} `json:"binding"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatal(err)
	}
	if status.Auth == nil || status.Auth.State != "needs_login" {
		t.Fatalf("auth = %+v, want needs_login", status.Auth)
	}
}
