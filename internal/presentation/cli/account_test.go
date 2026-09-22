package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
)

type cliEnv struct {
	app    *App
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func newCLIEnv(t *testing.T) *cliEnv {
	t.Helper()
	t.Setenv("GITRA_CONFIG_DIR", t.TempDir())
	application, err := bootstrap.New()
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return &cliEnv{app: New(application, stdout, stderr), stdout: stdout, stderr: stderr}
}

func (e *cliEnv) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	e.stdout.Reset()
	e.stderr.Reset()
	code := e.app.Execute(context.Background(), args)
	return code, e.stdout.String(), e.stderr.String()
}

func writeKey(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func (e *cliEnv) addAccount(t *testing.T, alias string) string {
	t.Helper()
	code, _, stderr := e.run(t, "account", "add",
		"--alias", alias, "--provider", "github", "--username", alias,
		"--name", "Luna", "--email", alias+"@example.com", "--key", writeKey(t, "id_ed25519_"+alias))
	if code != 0 {
		t.Fatalf("account add exit=%d stderr=%s", code, stderr)
	}
	return alias
}

func gitRunT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newRepo(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRunT(t, root, "init", "-q")
	gitRunT(t, root, "config", "user.name", "Old Name")
	gitRunT(t, root, "config", "user.email", "old@example.com")
	gitRunT(t, root, "remote", "add", "origin", "git@github.com:lunafoundry/luna-site.git")
	return root
}

func TestAccountAddListShowJSON(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")

	code, stdout, stderr := env.run(t, "account", "list", "--json")
	if code != 0 {
		t.Fatalf("list exit=%d stderr=%s", code, stderr)
	}
	var list struct {
		SchemaVersion int `json:"schema_version"`
		Accounts      []struct {
			ID       string `json:"id"`
			Alias    string `json:"alias"`
			Provider struct {
				Type string `json:"type"`
				Host string `json:"host"`
			} `json:"provider"`
			Auth struct {
				Strategy string `json:"strategy"`
				State    string `json:"state"`
			} `json:"auth"`
			ProjectCount int `json:"project_count"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("list JSON invalid: %v\n%s", err, stdout)
	}
	if list.SchemaVersion != 1 || len(list.Accounts) != 1 {
		t.Fatalf("list payload = %+v", list)
	}
	if list.Accounts[0].Provider.Host != "github.com" || list.Accounts[0].Auth.State != "configured" {
		t.Fatalf("account item = %+v", list.Accounts[0])
	}

	code, stdout, stderr = env.run(t, "account", "show", "luna", "--json")
	if code != 0 {
		t.Fatalf("show exit=%d stderr=%s", code, stderr)
	}
	var show struct {
		SchemaVersion int `json:"schema_version"`
		Account       struct {
			Alias string `json:"alias"`
		} `json:"account"`
		Projects []any `json:"projects"`
	}
	if err := json.Unmarshal([]byte(stdout), &show); err != nil {
		t.Fatalf("show JSON invalid: %v\n%s", err, stdout)
	}
	if show.Account.Alias != "luna" || show.Projects == nil {
		t.Fatalf("show payload = %+v", show)
	}
}

func TestAccountAddUsageErrors(t *testing.T) {
	env := newCLIEnv(t)
	if code, _, _ := env.run(t, "account", "add", "--alias", "luna"); code != 2 {
		t.Fatalf("missing flags exit = %d, want 2", code)
	}
	if code, _, _ := env.run(t, "account", "add", "--alias", "luna", "--provider", "bitbucket"); code != 2 {
		t.Fatalf("unknown provider exit = %d, want 2", code)
	}
	env.addAccount(t, "luna")
	code, _, stderr := env.run(t, "account", "add", "--alias", "luna", "--provider", "github",
		"--username", "luna", "--name", "Luna", "--email", "l@example.com", "--key", writeKey(t, "k"))
	if code != 3 || !strings.Contains(stderr, "already exists") {
		t.Fatalf("duplicate alias exit=%d stderr=%s, want 3", code, stderr)
	}
}

func TestAccountEditPropagatesToBoundRepository(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")
	repo := newRepo(t)

	accounts, err := env.app.App.Accounts.List(context.Background())
	if err != nil || len(accounts) != 1 {
		t.Fatalf("accounts = %v, %v", accounts, err)
	}
	if _, err := env.app.App.Bindings.Bind(context.Background(), app.BindRequest{AccountID: accounts[0].ID, Path: repo}); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := env.run(t, "account", "edit", "luna", "--email", "new@example.com")
	if code != 0 {
		t.Fatalf("edit exit=%d stderr=%s", code, stderr)
	}
	if got := gitRunT(t, repo, "config", "--local", "--get", "user.email"); got != "new@example.com" {
		t.Fatalf("bound repository not reconciled: user.email = %q", got)
	}
	if code, _, _ := env.run(t, "account", "edit", "luna"); code != 2 {
		t.Fatalf("edit without changes exit = %d, want 2", code)
	}
}

func TestAccountRemoveGuardsAndUnbindAll(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")
	repo := newRepo(t)
	accounts, _ := env.app.App.Accounts.List(context.Background())
	if _, err := env.app.App.Bindings.Bind(context.Background(), app.BindRequest{AccountID: accounts[0].ID, Path: repo}); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := env.run(t, "account", "remove", "luna")
	if code != 3 || !strings.Contains(stderr, "unbind-all") {
		t.Fatalf("remove bound exit=%d stderr=%s, want 3 with hint", code, stderr)
	}
	if code, _, stderr = env.run(t, "account", "remove", "luna", "--unbind-all"); code != 0 {
		t.Fatalf("remove --unbind-all exit=%d stderr=%s", code, stderr)
	}
	if _, _, stderr = env.run(t, "account", "show", "luna"); !strings.Contains(stderr, "not found") {
		t.Fatalf("account still present: %s", stderr)
	}
	if got := gitRunT(t, repo, "config", "--local", "--get", "user.email"); got != "old@example.com" {
		t.Fatalf("unbind did not restore: %q", got)
	}
}
