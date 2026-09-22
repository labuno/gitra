package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/presentation/cli"
)

type cliEnv struct {
	app    *cli.App
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
	return &cliEnv{app: cli.New(application, stdout, stderr), stdout: stdout, stderr: stderr}
}

func (e *cliEnv) run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	e.stdout.Reset()
	e.stderr.Reset()
	code := e.app.Execute(context.Background(), args)
	return code, e.stdout.String(), e.stderr.String()
}

func (e *cliEnv) addAccount(t *testing.T, alias string) {
	t.Helper()
	key := filepath.Join(t.TempDir(), "id_ed25519_"+alias)
	if err := os.WriteFile(key, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := e.run(t, "account", "add",
		"--alias", alias, "--provider", "github", "--username", alias,
		"--name", "Luna", "--email", alias+"@example.com", "--key", key); code != 0 {
		t.Fatalf("account add exit=%d stderr=%s", code, stderr)
	}
}

// TestCLIFullFlow walks the baseline §47 acceptance flow end to end.
func TestCLIFullFlow(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")
	repo := makeRepo(t, "flow-repo", "git@github.com:lunafoundry/flow.git")

	if code, _, stderr := env.run(t, "bind", "luna", repo); code != 0 {
		t.Fatalf("bind exit=%d stderr=%s", code, stderr)
	}

	code, stdout, stderr := env.run(t, "status", repo, "--json")
	if code != 0 {
		t.Fatalf("status exit=%d stderr=%s", code, stderr)
	}
	var status struct {
		SchemaVersion int    `json:"schema_version"`
		Repository    string `json:"repository"`
		Bound         bool   `json:"bound"`
		Binding       *struct {
			ID     string `json:"id"`
			Health string `json:"health"`
		} `json:"binding"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status JSON: %v\n%s", err, stdout)
	}
	if !status.Bound || status.Binding == nil || status.Binding.Health != "ok" || status.SchemaVersion != 1 {
		t.Fatalf("status payload = %+v", status)
	}

	if code, _, stderr := env.run(t, "account", "edit", "luna", "--email", "new@example.com"); code != 0 {
		t.Fatalf("edit exit=%d stderr=%s", code, stderr)
	}
	canonical := gitRun(t, repo, "rev-parse", "--show-toplevel")
	if got, _ := gitConfig(t, canonical, "user.email"); got != "new@example.com" {
		t.Fatalf("edit did not reconcile: user.email = %q", got)
	}

	if code, _, stderr := env.run(t, "status", repo); code != 0 {
		t.Fatalf("status after edit exit=%d stderr=%s", code, stderr)
	}

	if code, _, stderr := env.run(t, "unbind", repo); code != 0 {
		t.Fatalf("unbind exit=%d stderr=%s", code, stderr)
	}
	if got, _ := gitConfig(t, canonical, "user.email"); got != "old@example.com" {
		t.Fatalf("unbind did not restore: %q", got)
	}

	if code, _, stderr := env.run(t, "account", "remove", "luna"); code != 0 {
		t.Fatalf("remove exit=%d stderr=%s", code, stderr)
	}
	code, stdout, _ = env.run(t, "account", "list", "--json")
	if code != 0 {
		t.Fatalf("list exit=%d", code)
	}
	var list struct {
		Accounts []any `json:"accounts"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil || len(list.Accounts) != 0 {
		t.Fatalf("accounts remain: %s", stdout)
	}
}

// TestCLIExitCodeMatrix locks the exit-code contract from baseline §35.
func TestCLIExitCodeMatrix(t *testing.T) {
	env := newCLIEnv(t)

	if code, _, _ := env.run(t, "version"); code != 0 {
		t.Fatalf("success exit = %d, want 0", code)
	}
	if code, _, _ := env.run(t, "account", "add", "--alias", "x"); code != 2 {
		t.Fatalf("missing flags exit = %d, want 2", code)
	}
	if code, _, _ := env.run(t, "account", "show", "ghost"); code != 3 {
		t.Fatalf("unknown account exit = %d, want 3", code)
	}
	env.addAccount(t, "luna")
	if code, _, _ := env.run(t, "bind", "luna", t.TempDir()); code != 4 {
		t.Fatalf("non-repository exit = %d, want 4", code)
	}
	if code, _, _ := env.run(t, "account", "add", "--alias", "broken", "--provider", "github",
		"--username", "broken", "--name", "B", "--email", "b@example.com", "--key", "/nonexistent/key"); code != 5 {
		t.Fatalf("invalid key exit = %d, want 5", code)
	}
	repo := makeRepo(t, "matrix-repo", "git@github.com:lunafoundry/matrix.git")
	if code, _, _ := env.run(t, "bind", "luna", repo); code != 0 {
		t.Fatalf("bind exit = %d", code)
	}
	if code, _, _ := env.run(t, "bind", "luna", repo); code != 6 {
		t.Fatalf("already bound exit = %d, want 6", code)
	}
	httpsRepo := makeRepo(t, "matrix-https", "https://github.com/lunafoundry/matrix.git")
	if code, _, _ := env.run(t, "bind", "luna", httpsRepo); code != 7 {
		t.Fatalf("unsupported remote exit = %d, want 7", code)
	}
}
