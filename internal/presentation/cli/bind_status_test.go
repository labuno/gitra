package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBindUnbindFlow(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")
	repo := newRepo(t)

	code, stdout, stderr := env.run(t, "bind", "luna", repo)
	if code != 0 {
		t.Fatalf("bind exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "bound ") {
		t.Fatalf("bind stdout = %q", stdout)
	}
	if got := gitRunT(t, repo, "config", "--local", "--get", "user.email"); got != "luna@example.com" {
		t.Fatalf("user.email = %q", got)
	}

	code, _, stderr = env.run(t, "bind", "luna", repo)
	if code != 6 || !strings.Contains(stderr, "already bound") {
		t.Fatalf("duplicate bind exit=%d stderr=%s, want 6", code, stderr)
	}

	if code, _, stderr = env.run(t, "unbind", repo); code != 0 {
		t.Fatalf("unbind exit=%d stderr=%s", code, stderr)
	}
	if got := gitRunT(t, repo, "config", "--local", "--get", "user.email"); got != "old@example.com" {
		t.Fatalf("unbind did not restore: %q", got)
	}
}

func TestBindErrors(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")

	if code, _, _ := env.run(t, "bind", "missing", newRepo(t)); code != 3 {
		t.Fatalf("unknown alias exit = %d, want 3", code)
	}
	if code, _, _ := env.run(t, "bind", "luna", t.TempDir()); code != 4 {
		t.Fatalf("non-repo exit = %d, want 4", code)
	}

	httpsRepo := filepath.Join(t.TempDir(), "https-repo")
	if err := os.MkdirAll(httpsRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRunT(t, httpsRepo, "init", "-q")
	gitRunT(t, httpsRepo, "remote", "add", "origin", "https://github.com/lunafoundry/luna-site.git")
	code, _, stderr := env.run(t, "bind", "luna", httpsRepo)
	if code != 7 {
		t.Fatalf("https remote exit = %d stderr=%s, want 7", code, stderr)
	}
	if !strings.Contains(stderr, "SSH") {
		t.Fatalf("https error must explain the SSH requirement: %s", stderr)
	}
}

func TestBindUsesCurrentDirectoryByDefault(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")
	repo := newRepo(t)
	t.Chdir(repo)

	if code, _, stderr := env.run(t, "bind", "luna"); code != 0 {
		t.Fatalf("bind default path exit=%d stderr=%s", code, stderr)
	}
	if got := gitRunT(t, repo, "config", "--local", "--get", "gitra.bindingId"); got == "" {
		t.Fatal("metadata missing after bind")
	}
}

func TestStatusJSONAndHealth(t *testing.T) {
	env := newCLIEnv(t)
	env.addAccount(t, "luna")
	repo := newRepo(t)

	code, stdout, stderr := env.run(t, "status", repo, "--json")
	if code != 0 {
		t.Fatalf("unbound status exit=%d stderr=%s", code, stderr)
	}
	var unbound struct {
		SchemaVersion int    `json:"schema_version"`
		Repository    string `json:"repository"`
		Bound         bool   `json:"bound"`
	}
	if err := json.Unmarshal([]byte(stdout), &unbound); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	canonical := gitRunT(t, repo, "rev-parse", "--show-toplevel")
	if unbound.Bound || unbound.Repository != canonical {
		t.Fatalf("unbound payload = %+v, want canonical repository %q", unbound, canonical)
	}

	if code, _, stderr = env.run(t, "bind", "luna", repo); code != 0 {
		t.Fatalf("bind exit=%d stderr=%s", code, stderr)
	}

	code, stdout, stderr = env.run(t, "status", repo, "--json")
	if code != 0 {
		t.Fatalf("bound status exit=%d stderr=%s", code, stderr)
	}
	var bound struct {
		SchemaVersion int    `json:"schema_version"`
		Repository    string `json:"repository"`
		Bound         bool   `json:"bound"`
		Binding       *struct {
			ID        string `json:"id"`
			AccountID string `json:"account_id"`
			Strategy  string `json:"strategy"`
			Health    string `json:"health"`
		} `json:"binding"`
		Account *struct {
			Alias string `json:"alias"`
		} `json:"account"`
		Auth *struct {
			Strategy string `json:"strategy"`
			State    string `json:"state"`
		} `json:"auth"`
	}
	if err := json.Unmarshal([]byte(stdout), &bound); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if !bound.Bound || bound.Binding == nil || bound.Binding.Health != "ok" {
		t.Fatalf("bound payload = %+v", bound)
	}
	if bound.Account == nil || bound.Account.Alias != "luna" || bound.Auth == nil || bound.Auth.State != "configured" {
		t.Fatalf("bound payload account/auth = %+v", bound)
	}

	// inject drift
	gitRunT(t, repo, "config", "--local", "user.email", "drifted@example.com")
	code, stdout, _ = env.run(t, "status", repo, "--json")
	if code != 6 {
		t.Fatalf("drift status exit=%d, want 6", code)
	}
	var drifted struct {
		Binding *struct {
			Health string `json:"health"`
		} `json:"binding"`
	}
	if err := json.Unmarshal([]byte(stdout), &drifted); err != nil {
		t.Fatal(err)
	}
	if drifted.Binding == nil || drifted.Binding.Health != "drift" {
		t.Fatalf("drift payload = %+v", drifted)
	}
}
