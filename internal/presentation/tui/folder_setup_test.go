package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
)

func TestPlainFolderCanInitializeAssociateAndBind(t *testing.T) {
	model, application := newTestModel(t)
	account := addHTTPSAccount(t, application, "luna")
	path := t.TempDir()
	model.startBind(account, path)
	model, _ = press(t, model, "b")
	if model.screen != screenRemote || model.errText != "" {
		t.Fatalf("plain folder should enter remote setup: screen=%v error=%s", model.screen, model.errText)
	}
	if model.bind.remoteURL != "https://github.com/luna/"+filepath.Base(path)+".git" {
		t.Fatalf("remote suggestion = %q", model.bind.remoteURL)
	}
	if got := strings.TrimSpace(gitRunT(t, path, "symbolic-ref", "--short", "HEAD")); got != "main" {
		t.Fatalf("initial branch = %s", got)
	}
	verify := exec.Command("git", "-C", path, "rev-parse", "--verify", "HEAD")
	if err := verify.Run(); err == nil {
		t.Fatal("initialization must not create a commit")
	}
	if err := application.Bindings.EnsureOriginRemote(context.Background(), path, "https://github.com/luna/project.git"); err != nil {
		t.Fatal(err)
	}
	model, cmd := press(t, model, "enter")
	// The existing provider lookup is tested separately; simulate its result.
	if cmd == nil {
		t.Fatal("remote setup should start a lookup")
	}
	updated, bind := model.Update(remoteCheckMsg{check: app.RemoteCheck{Exists: true}})
	model = updated.(Model)
	model = runCmd(t, model, bind)
	status, err := application.Bindings.Status(context.Background(), path)
	if err != nil || !status.Bound || status.Account.ID != account.ID {
		t.Fatalf("binding = %+v, error=%v", status, err)
	}
	if got := strings.TrimSpace(gitRunT(t, path, "status", "--porcelain")); got != "" {
		t.Fatalf("initialization should not stage files: %s", got)
	}
}

func TestPrepareRepositoryPreservesExistingRepoAndRejectsUnsupported(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "user.useConfigOnly")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")
	_, application := newTestModel(t)
	account := addHTTPSAccount(t, application, "luna")
	path := newRepoDir(t)
	gitRunT(t, path, "-c", "user.name=Test User", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "existing")
	head := gitRunT(t, path, "rev-parse", "HEAD")
	origin := gitRunT(t, path, "remote", "get-url", "origin")
	if _, err := application.Bindings.PrepareRepository(context.Background(), account.ID, path); err != nil {
		t.Fatal(err)
	}
	if head != gitRunT(t, path, "rev-parse", "HEAD") || origin != gitRunT(t, path, "remote", "get-url", "origin") {
		t.Fatal("existing history and remote must be preserved")
	}
	bare := t.TempDir()
	gitRunT(t, bare, "init", "--bare")
	if _, err := application.Bindings.PrepareRepository(context.Background(), account.ID, bare); err == nil {
		t.Fatal("bare repositories must remain unsupported")
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := application.Bindings.PrepareRepository(context.Background(), account.ID, missing); err == nil {
		t.Fatal("missing directories must not be initialized")
	}
	plain := t.TempDir()
	if _, err := application.Bindings.PrepareRepository(context.Background(), domain.AccountID("acc_missing"), plain); err == nil {
		t.Fatal("invalid account must fail before initialization")
	}
	if _, err := os.Stat(filepath.Join(plain, ".git")); !os.IsNotExist(err) {
		t.Fatal("invalid account initialized a folder")
	}
}

func TestPickerStartDirectorySaveFailureIsVisible(t *testing.T) {
	model, application := newTestModel(t)
	account := addHTTPSAccount(t, application, "luna")
	model.startBind(account, t.TempDir())
	if err := os.Mkdir(filepath.Join(os.Getenv("GITRA_CONFIG_DIR"), "picker.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	model, _ = press(t, model, "s")
	if model.errText == "" || model.message != "" {
		t.Fatalf("save error hidden: error=%q message=%q", model.errText, model.message)
	}
}

func TestInitializationFailureIsVisibleAndDoesNotBind(t *testing.T) {
	model, application := newTestModel(t)
	account := addHTTPSAccount(t, application, "luna")
	application.Deps.Initializer = nil
	application.Bindings = app.NewBindingService(application.Deps)
	path := t.TempDir()
	model.startBind(account, path)
	model, cmd := press(t, model, "b")
	if cmd != nil || model.screen != screenBind || model.errText == "" {
		t.Fatalf("initialization error hidden: screen=%v error=%q", model.screen, model.errText)
	}
	if _, err := os.Stat(filepath.Join(path, ".git")); !os.IsNotExist(err) {
		t.Fatal("failed initialization should not produce a repository")
	}
}

func TestCancelRemoteSetupLeavesUnboundRepository(t *testing.T) {
	model, application := newTestModel(t)
	account := addHTTPSAccount(t, application, "luna")
	path := t.TempDir()
	model.startBind(account, path)
	model, _ = press(t, model, "b")
	model, _ = press(t, model, "esc")
	status, err := application.Bindings.Status(context.Background(), path)
	if err != nil || status.Bound || status.HasOrigin || model.screen != screenBind {
		t.Fatalf("cancelled setup = %+v, error=%v", status, err)
	}
}

func TestPlainFolderSuggestsSSHRemote(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	account, _ := application.Accounts.GetByAlias(context.Background(), "luna")
	path := t.TempDir()
	model.startBind(account, path)
	model, _ = press(t, model, "b")
	want := "git@github.com:luna/" + filepath.Base(path) + ".git"
	if model.screen != screenRemote || model.bind.remoteURL != want {
		t.Fatalf("SSH suggestion = %q, error=%s", model.bind.remoteURL, model.errText)
	}
	model, cmd := press(t, model, "enter")
	if cmd == nil {
		t.Fatal("SSH association must produce a command")
	}
	model = runCmd(t, model, cmd)
	status, err := application.Bindings.Status(context.Background(), path)
	if err != nil || !status.Bound || status.OriginURL != want {
		t.Fatalf("SSH binding = %+v, error=%v, ui=%s", status, err, model.errText)
	}
}

func TestPickerStartDirectorySurvivesRestart(t *testing.T) {
	model, application := newTestModel(t)
	account := addHTTPSAccount(t, application, "luna")
	base := t.TempDir()
	chosen := filepath.Join(base, "projects")
	if err := os.Mkdir(chosen, 0o755); err != nil {
		t.Fatal(err)
	}
	model.startBind(account, base)
	model = descendTo(t, model, "projects")
	model, _ = press(t, model, "s")
	if !strings.Contains(model.message, "起始目录") {
		t.Fatalf("save should be visible: %s", model.message)
	}
	second, err := bootstrap.New()
	if err != nil {
		t.Fatal(err)
	}
	restarted := New(second)
	restarted.lastBindDir = base
	restarted.startBind(account, base)
	if restarted.bind.path != chosen || restarted.bind.startPath != chosen {
		t.Fatalf("saved start directory lost: %+v", restarted.bind)
	}
	if err := os.Remove(chosen); err != nil {
		t.Fatal(err)
	}
	restarted = New(second)
	restarted.startBind(account, base)
	if restarted.screen == screenBind || restarted.errText == "" {
		t.Fatal("a removed start directory should report an error")
	}
}
