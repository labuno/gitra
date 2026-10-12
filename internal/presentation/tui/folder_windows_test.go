//go:build windows

package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/app"
)

func TestWindowsProjectLifecycleWithUnicodePaths(t *testing.T) {
	model, application := newTestModel(t)
	account := addHTTPSAccount(t, application, "luna")
	path := filepath.Join(t.TempDir(), "项目 with spaces")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	model.startBind(account, path)
	model, _ = press(t, model, "b")
	if model.screen != screenRemote || model.errText != "" {
		t.Fatalf("initialize project: %s", model.errText)
	}
	if err := application.Bindings.EnsureOriginRemote(context.Background(), path, "https://github.com/luna/project.git"); err != nil {
		t.Fatal(err)
	}
	updated, bind := model.Update(remoteCheckMsg{check: app.RemoteCheck{Exists: true}})
	model = runCmd(t, updated.(Model), bind)
	if model.errText != "" {
		t.Fatal(model.errText)
	}
	bare := filepath.Join(t.TempDir(), "远端 repo.git")
	gitRunT(t, filepath.Dir(bare), "init", "--bare", bare)
	repo, err := application.Deps.Git.DiscoverRepository(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.Deps.Git.SetLocalConfig(context.Background(), repo, "remote.origin.url", filepath.ToSlash(bare)); err != nil {
		t.Fatal(err)
	}
	gitRunT(t, path, "commit", "--allow-empty", "-m", "initial")
	if _, err := application.Publish.FirstPublish(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(gitRunT(t, path, "rev-parse", "--abbrev-ref", "@{upstream}")); got != "origin/main" {
		t.Fatalf("upstream = %q", got)
	}
	if err := application.Bindings.Unbind(context.Background(), app.UnbindRequest{Path: path}); err != nil {
		t.Fatal(err)
	}
	if _, present := gitConfigOrEmpty(t, path, "gitra.bindingId"); present {
		t.Fatal("unbind left binding metadata")
	}
}
