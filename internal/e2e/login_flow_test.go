package e2e

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/gitcli"
	"github.com/zhanhd/gitra/internal/adapters/provider/router"
	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/adapters/secretstore"
	"github.com/zhanhd/gitra/internal/adapters/storage"
	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/presentation/cli"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/httptoken"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"
)

// ghBlockingRunner keeps E2E deterministic: gh/glab reuse is disabled.
type ghBlockingRunner struct{ real ports.CommandRunner }

func (r ghBlockingRunner) Run(ctx context.Context, name string, args ...string) (ports.ProcessResult, error) {
	if name == "gh" || name == "glab" {
		return ports.ProcessResult{ExitCode: 127}, nil
	}
	return r.real.Run(ctx, name, args...)
}

func (r ghBlockingRunner) RunWithInput(ctx context.Context, name, input string, args ...string) (ports.ProcessResult, error) {
	if name == "gh" || name == "glab" {
		return ports.ProcessResult{ExitCode: 127}, nil
	}
	return r.real.RunWithInput(ctx, name, input, args...)
}

func githubAPIServer(t *testing.T, unauthorized bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	status := http.StatusOK
	if unauthorized {
		status = http.StatusUnauthorized
	}
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"login":"lunafoundry","name":"Luna"}`))
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`[{"email":"luna@example.com","primary":true,"verified":true}]`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

type loginE2E struct {
	cli         *cli.App
	application *bootstrap.App
	deps        app.Deps
	configDir   string
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
}

func newLoginE2E(t *testing.T, server *httptest.Server, stdin string) *loginE2E {
	t.Helper()
	configDir := t.TempDir()
	t.Setenv("GITRA_CONFIG_DIR", configDir)
	t.Setenv("GITRA_TOKEN", "")

	helper := "store --file=" + filepath.Join(configDir, "credentials")
	gitRunner := ghBlockingRunner{real: runner.New()}
	secrets := secretstore.NewGitCredentialStore(gitRunner, helper)

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
		CredentialHelpers: fixedHelperResolver{spec: helper},
	}
	loginAdapter := router.New(gitRunner, strings.NewReader(stdin))
	loginAdapter.BaseURL = func(domain.ProviderType, string) string { return server.URL }

	application := bootstrap.NewFromDeps(deps, loginAdapter, loginAdapter)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return &loginE2E{
		cli:         cli.New(application, stdout, stderr),
		application: application,
		deps:        deps,
		configDir:   configDir,
		stdout:      stdout,
		stderr:      stderr,
	}
}

type fixedHelperResolver struct{ spec string }

func (f fixedHelperResolver) Helper(context.Context) (string, bool, error) { return f.spec, true, nil }

func (e *loginE2E) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	e.stdout.Reset()
	e.stderr.Reset()
	code := e.cli.Execute(context.Background(), args)
	return code, e.stdout.String(), e.stderr.String()
}

func TestLoginBindAndCredentialRetrieval(t *testing.T) {
	server := githubAPIServer(t, false)
	env := newLoginE2E(t, server, "gho_e2e_token\n")
	ctx := context.Background()

	code, stdout, stderr := env.run(t, "login", "--provider", "github", "--token-stdin")
	if code != 0 {
		t.Fatalf("login exit=%d stderr=%s", code, stderr)
	}
	if strings.Contains(stdout+stderr, "gho_e2e_token") {
		t.Fatal("token leaked into output")
	}
	accounts, err := env.application.Accounts.List(ctx)
	if err != nil || len(accounts) != 1 {
		t.Fatalf("accounts = %v, %v", accounts, err)
	}
	account := accounts[0]
	if account.Alias != "lunafoundry" || account.Identity.Email != "luna@example.com" ||
		account.Transport.Strategy != domain.StrategyHTTPSToken {
		t.Fatalf("account = %+v", account)
	}

	repo := makeRepo(t, "site", "https://github.com/lunafoundry/site.git")
	if _, err := env.application.Bindings.Bind(ctx, app.BindRequest{AccountID: account.ID, Path: repo}); err != nil {
		t.Fatalf("bind error = %v", err)
	}
	canonical := gitRun(t, repo, "rev-parse", "--show-toplevel")
	if got, _ := gitConfig(t, canonical, "credential.https://github.com.username"); got != "lunafoundry" {
		t.Fatalf("credential scope = %q", got)
	}

	// native git must be able to fetch the token through the projected helper.
	helperSpec, _ := gitConfig(t, canonical, "credential.helper")
	if !strings.HasPrefix(helperSpec, "store --file=") {
		t.Fatalf("helper = %q", helperSpec)
	}
	fill := gitCredentialFill(t, canonical, helperSpec, "github.com", "lunafoundry")
	if fill != "gho_e2e_token" {
		t.Fatalf("credential fill = %q, want the stored token", fill)
	}

	if code, _, stderr := env.run(t, "unbind", repo); code != 0 {
		t.Fatalf("unbind exit=%d stderr=%s", code, stderr)
	}
	if got, _ := gitConfig(t, canonical, "credential.https://github.com.username"); got != "" {
		t.Fatalf("credential scope left after unbind: %q", got)
	}
	if got, _ := gitConfig(t, canonical, "user.email"); got != "old@example.com" {
		t.Fatalf("identity not restored: %q", got)
	}

	if code, _, stderr := env.run(t, "logout", "lunafoundry"); code != 0 {
		t.Fatalf("logout exit=%d stderr=%s", code, stderr)
	}
	if _, err := env.deps.Secrets.Get(ctx, "github.com/lunafoundry"); err == nil {
		t.Fatal("credential still present after logout")
	}
}

func TestLoginErrorPathsAndSecretHygiene(t *testing.T) {
	server := githubAPIServer(t, false)
	env := newLoginE2E(t, server, "")

	code, _, stderr := env.run(t, "login", "--provider", "github")
	if code != 5 {
		t.Fatalf("login without credential exit=%d stderr=%s, want 5", code, stderr)
	}
	for _, want := range []string{"--token-stdin", "GITRA_TOKEN"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("guidance missing %q: %s", want, stderr)
		}
	}

	badServer := githubAPIServer(t, true)
	rejectEnv := newLoginE2E(t, badServer, "gho_bad_token\n")
	if code, _, _ := rejectEnv.run(t, "login", "--provider", "github", "--token-stdin"); code != 5 {
		t.Fatalf("401 login exit=%d, want 5", code)
	}

	goodEnv := newLoginE2E(t, server, "gho_scan_token\n")
	if code, _, stderr := goodEnv.run(t, "login", "--provider", "github", "--token-stdin"); code != 0 {
		t.Fatalf("login exit=%d stderr=%s", code, stderr)
	}
	// Hygiene: the token may only live in the credential store itself.
	_ = filepath.WalkDir(goodEnv.configDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "credentials" {
			info, statErr := entry.Info()
			if statErr == nil && info.Mode().Perm() != 0o600 {
				t.Errorf("credential file permissions = %o, want 600", info.Mode().Perm())
			}
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Contains(data, []byte("gho_scan_token")) {
			t.Errorf("token leaked into %s", path)
		}
		return nil
	})
}

// gitCredentialFill asks git (with the repository's helper) for the stored
// credential, exactly like a push would.
func gitCredentialFill(t *testing.T, dir, helper, host, username string) string {
	t.Helper()
	cmd := execCommand(t, dir, "git", "-c", "credential.helper="+helper, "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=" + host + "\nusername=" + username + "\n\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git credential fill: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "password="); ok {
			return value
		}
	}
	return ""
}

func execCommand(t *testing.T, dir string, name string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd
}
