package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/gitcli"
	"github.com/zhanhd/gitra/internal/adapters/provider/github"
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

type stubVerifyProfiles struct {
	profile ports.ProviderProfile
	err     error
}

func (s stubVerifyProfiles) Profile(context.Context, domain.ProviderType, string, string) (ports.ProviderProfile, error) {
	return s.profile, s.err
}

type stubSSH struct {
	result ports.SSHTestResult
	err    error
}

func (s stubSSH) Test(context.Context, ports.SSHTestRequest) (ports.SSHTestResult, error) {
	return s.result, s.err
}

type sshParserFunc func(string, string) (string, bool)

func (f sshParserFunc) ParseSSHIdentity(_ context.Context, _ domain.ProviderType, stdout, stderr string) (string, bool) {
	return f(stdout, stderr)
}

type verifyCLIEnv struct {
	cli         *App
	application *bootstrap.App
	deps        app.Deps
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
}

func newVerifyCLIEnv(t *testing.T, profiles ports.ProfileProvider, ssh ports.SSH, parser ports.SSHIdentityParser) *verifyCLIEnv {
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
		SSH:               ssh,
	}
	application := bootstrap.NewFromDeps(deps, nil, profiles, parser)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return &verifyCLIEnv{cli: New(application, stdout, stderr), application: application, deps: deps, stdout: stdout, stderr: stderr}
}

func (e *verifyCLIEnv) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	e.stdout.Reset()
	e.stderr.Reset()
	code := e.cli.Execute(context.Background(), args)
	return code, e.stdout.String(), e.stderr.String()
}

func (e *verifyCLIEnv) createHTTPSAccount(t *testing.T, token string) domain.Account {
	t.Helper()
	// The credential must exist before the account is created: account
	// validation checks local auth material.
	if token != "" {
		if err := e.deps.Secrets.Set(context.Background(), "github.com/lunafoundry", token); err != nil {
			t.Fatal(err)
		}
	}
	account, err := e.application.Accounts.Create(context.Background(), app.CreateAccountRequest{
		Alias: "luna",
		Provider: domain.ProviderRef{
			Type: domain.ProviderGitHub, Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategyHTTPSToken, Config: map[string]string{}},
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	return account
}

func TestAccountTestHTTPSOK(t *testing.T) {
	env := newVerifyCLIEnv(t, stubVerifyProfiles{profile: ports.ProviderProfile{Username: "lunafoundry"}}, stubSSH{}, nil)
	env.createHTTPSAccount(t, "tok")

	code, stdout, stderr := env.run(t, "account", "test", "luna")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "authenticated") {
		t.Fatalf("stdout = %q", stdout)
	}

	code, stdout, _ = env.run(t, "account", "test", "luna", "--json")
	if code != 0 {
		t.Fatalf("json exit=%d", code)
	}
	var payload struct {
		SchemaVersion    int    `json:"schema_version"`
		Status           string `json:"status"`
		Success          bool   `json:"success"`
		ExpectedUsername string `json:"expected_username"`
		ActualUsername   string `json:"actual_username"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json invalid: %v\n%s", err, stdout)
	}
	if payload.SchemaVersion != 1 || !payload.Success || payload.Status != "ok" ||
		payload.ExpectedUsername != "lunafoundry" || payload.ActualUsername != "lunafoundry" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestAccountTestHTTPSFailureClasses(t *testing.T) {
	mismatch := newVerifyCLIEnv(t, stubVerifyProfiles{profile: ports.ProviderProfile{Username: "someone-else"}}, stubSSH{}, nil)
	mismatch.createHTTPSAccount(t, "tok")
	code, _, stderr := mismatch.run(t, "account", "test", "luna")
	if code != 5 || !strings.Contains(stderr, "someone-else") {
		t.Fatalf("mismatch exit=%d stderr=%s, want 5", code, stderr)
	}

	rejected := newVerifyCLIEnv(t, stubVerifyProfiles{err: fmt.Errorf("%w: 401", domain.ErrAuthInvalid)}, stubSSH{}, nil)
	rejected.createHTTPSAccount(t, "tok")
	if code, _, _ := rejected.run(t, "account", "test", "luna"); code != 5 {
		t.Fatalf("rejected exit=%d, want 5", code)
	}

	unavailable := newVerifyCLIEnv(t, stubVerifyProfiles{err: errors.New("provider returned HTTP 500")}, stubSSH{}, nil)
	unavailable.createHTTPSAccount(t, "tok")
	if code, _, _ := unavailable.run(t, "account", "test", "luna"); code != 1 {
		t.Fatalf("unavailable exit=%d, want 1", code)
	}

	noCredential := newVerifyCLIEnv(t, stubVerifyProfiles{}, stubSSH{}, nil)
	noCredential.createHTTPSAccount(t, "tok")
	_ = noCredential.deps.Secrets.Delete(context.Background(), "github.com/lunafoundry")
	code, _, stderr = noCredential.run(t, "account", "test", "luna")
	if code != 5 || !strings.Contains(stderr, "gitra login") {
		t.Fatalf("missing credential exit=%d stderr=%s, want 5 with login hint", code, stderr)
	}
}

func TestAccountTestSSHPaths(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "id_ed25519_luna")
	if err := os.WriteFile(keyPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}

	okEnv := newVerifyCLIEnv(t, stubVerifyProfiles{},
		stubSSH{result: ports.SSHTestResult{ExitCode: 1, Stderr: "Hi lunafoundry! You've successfully authenticated, but GitHub does not provide shell access."}},
		sshParserFunc(github.ParseSSHVerification))
	_, err := okEnv.application.Accounts.Create(context.Background(), app.CreateAccountRequest{
		Alias: "ssh-luna",
		Provider: domain.ProviderRef{
			Type: domain.ProviderGitHub, Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": keyPath}},
	})
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := okEnv.run(t, "account", "test", "ssh-luna", "--verbose")
	if code != 0 {
		t.Fatalf("ssh ok exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "raw: ") {
		t.Fatalf("verbose output must include raw ssh output: %q", stdout)
	}

	deniedEnv := newVerifyCLIEnv(t, stubVerifyProfiles{},
		stubSSH{result: ports.SSHTestResult{ExitCode: 255, Stderr: "git@github.com: Permission denied (publickey)."}},
		sshParserFunc(github.ParseSSHVerification))
	_, err = deniedEnv.application.Accounts.Create(context.Background(), app.CreateAccountRequest{
		Alias: "ssh-luna",
		Provider: domain.ProviderRef{
			Type: domain.ProviderGitHub, Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": keyPath}},
	})
	if err != nil {
		t.Fatal(err)
	}
	code, _, stderr = deniedEnv.run(t, "account", "test", "ssh-luna")
	if code != 5 || !strings.Contains(stderr, "could not recognize") {
		t.Fatalf("denied exit=%d stderr=%s, want 5 with unrecognized hint", code, stderr)
	}
}

func TestAccountTestUsageErrors(t *testing.T) {
	env := newVerifyCLIEnv(t, stubVerifyProfiles{}, stubSSH{}, nil)
	if code, _, _ := env.run(t, "account", "test"); code != 2 {
		t.Fatalf("missing arg exit = %d, want 2", code)
	}
	if code, _, _ := env.run(t, "account", "test", "ghost"); code != 3 {
		t.Fatalf("unknown alias exit = %d, want 3", code)
	}
}
