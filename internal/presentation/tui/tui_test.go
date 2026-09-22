package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
)

func newTestModel(t *testing.T) (Model, *bootstrap.App) {
	t.Helper()
	t.Setenv("GITRA_CONFIG_DIR", t.TempDir())
	application, err := bootstrap.New()
	if err != nil {
		t.Fatal(err)
	}
	model := New(application)
	model.width, model.height = 120, 40
	return model, application
}

func addSSHAccount(t *testing.T, application *bootstrap.App, alias string) {
	t.Helper()
	keyPath := filepath.Join(t.TempDir(), "id_ed25519_"+alias)
	if err := os.WriteFile(keyPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := application.Accounts.Create(context.Background(), app.CreateAccountRequest{
		Alias: alias,
		Provider: domain.ProviderRef{
			Type: domain.ProviderGitHub, Username: alias,
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: alias, Email: alias + "@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": keyPath}},
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
}

func newRepoDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", "https://github.com/luna/site.git"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func press(t *testing.T, model Model, key string) (Model, tea.Cmd) {
	t.Helper()
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, cmd := model.Update(msg)
	return updated.(Model), cmd
}

func TestColumnsFor(t *testing.T) {
	tests := []struct {
		width int
		want  int
	}{
		{30, 1},
		{41, 1},
		{80, 1},
		{160, 3},
		{240, 4},
	}
	for _, tt := range tests {
		if got := columnsFor(tt.width); got != tt.want {
			t.Errorf("columnsFor(%d) = %d, want %d", tt.width, got, tt.want)
		}
	}
}

func TestLoginWizardStepsAndMasking(t *testing.T) {
	model, _ := newTestModel(t)
	model, _ = press(t, model, "a")
	if model.screen != screenLogin || model.login.step != 0 {
		t.Fatalf("screen=%v step=%d", model.screen, model.login.step)
	}
	model, _ = press(t, model, "down")
	if model.login.providerIndex != 1 {
		t.Fatalf("providerIndex = %d, want 1", model.login.providerIndex)
	}
	model, _ = press(t, model, "enter")
	if model.login.step != 1 {
		t.Fatalf("step = %d, want method selection", model.login.step)
	}
	model, _ = press(t, model, "down")
	model, _ = press(t, model, "enter")
	if model.login.step != 2 {
		t.Fatalf("step = %d, want token input", model.login.step)
	}
	for _, key := range []string{"z", "z", "s", "e", "c"} {
		model, _ = press(t, model, key)
	}
	if model.login.token != "zzsec" {
		t.Fatalf("token = %q", model.login.token)
	}
	if view := model.View(); strings.Contains(view, "zzsec") || !strings.Contains(view, "•") {
		t.Fatalf("token must be masked in the view:\n%s", view)
	}
	model, _ = press(t, model, "esc")
	if model.login.step != 1 {
		t.Fatalf("esc must go back one step, got %d", model.login.step)
	}
	model, _ = press(t, model, "esc")
	model, _ = press(t, model, "esc")
	if model.screen != screenAccounts {
		t.Fatalf("screen = %v, want accounts", model.screen)
	}
}

func TestEmptyStateGuidesToLogin(t *testing.T) {
	model, _ := newTestModel(t)
	view := model.View()
	if !strings.Contains(view, "欢迎使用 gitra") || !strings.Contains(view, "按 A 登录") {
		t.Fatalf("empty state must guide the user to log in:\n%s", view)
	}
	model, _ = press(t, model, "enter")
	if model.screen != screenLogin {
		t.Fatalf("Enter on the welcome screen should start login, got %v", model.screen)
	}
}

func TestAccountsNavigationAndDetail(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	addSSHAccount(t, application, "work")
	model.loadAccounts()
	if len(model.accounts) != 2 {
		t.Fatalf("accounts = %+v", model.accounts)
	}
	model.width = 160 // three columns
	model, _ = press(t, model, "right")
	if model.selected != 1 {
		t.Fatalf("selected = %d, want 1", model.selected)
	}
	model, _ = press(t, model, "left")
	if model.selected != 0 {
		t.Fatalf("selected = %d, want 0", model.selected)
	}

	view := model.View()
	if !strings.Contains(view, "luna") || !strings.Contains(view, "github.com") || !strings.Contains(view, "可用") {
		t.Fatalf("card view missing account facts:\n%s", view)
	}

	model, _ = press(t, model, "enter")
	if model.screen != screenDetail || model.detailAccount == nil {
		t.Fatalf("detail not opened: screen=%v", model.screen)
	}
	detailView := model.View()
	if !strings.Contains(detailView, "提交身份") || !strings.Contains(detailView, "已绑定的项目") {
		t.Fatalf("detail view incomplete:\n%s", detailView)
	}
	model, _ = press(t, model, "esc")
	if model.screen != screenAccounts {
		t.Fatalf("esc must return to accounts, got %v", model.screen)
	}
}

func TestBindPickerNavigation(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	model.openDetail()
	repo := newRepoDir(t)
	model.startBind(*model.detailAccount, repo)
	if model.screen != screenBind || model.bind.path != repo {
		t.Fatalf("bind state = %+v", model.bind)
	}
	view := model.View()
	if !strings.Contains(view, "使用这个文件夹") || !strings.Contains(view, repo) {
		t.Fatalf("bind view incomplete:\n%s", view)
	}
	model, _ = press(t, model, "e")
	if !model.bind.manual {
		t.Fatal("e should switch to manual path input")
	}
	model, _ = press(t, model, "esc")
}

func TestOnboardingOfferAfterFirstLogin(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	repo := newRepoDir(t)
	t.Chdir(repo)
	model.loadAccounts()

	updated, _ := model.Update(loginDoneMsg{alias: "luna", username: "lunafoundry", host: "github.com", created: true})
	model = updated.(Model)
	if model.screen != screenConfirm || !strings.Contains(model.confirmPrompt, "绑定") {
		t.Fatalf("expected onboarding bind offer, got screen=%v prompt=%q", model.screen, model.confirmPrompt)
	}
	if !strings.Contains(model.confirmPrompt, repo) {
		t.Fatalf("prompt must mention the repository path: %q", model.confirmPrompt)
	}
}

func TestConfirmDialogRunsAction(t *testing.T) {
	model, _ := newTestModel(t)
	ran := false
	model.screen = screenConfirm
	model.confirmPrompt = "测试"
	model.confirmAction = func() tea.Cmd {
		ran = true
		return nil
	}
	model, _ = press(t, model, "y")
	if !ran {
		t.Fatal("confirming must run the action")
	}
	if model.screen != screenAccounts {
		t.Fatalf("screen = %v, want accounts", model.screen)
	}

	ran = false
	model.screen = screenConfirm
	model.confirmAction = func() tea.Cmd { ran = true; return nil }
	model, _ = press(t, model, "n")
	if ran {
		t.Fatal("cancelling must not run the action")
	}
}

func TestVerifyMessageRendering(t *testing.T) {
	model, _ := newTestModel(t)
	updated, _ := model.Update(verifyDoneMsg{success: false, status: "authentication_failed", message: "登录已失效"})
	model = updated.(Model)
	if !strings.Contains(model.View(), "连接检查未通过") {
		t.Fatalf("failure must be visible:\n%s", model.View())
	}
}

func TestPlainErrorTranslation(t *testing.T) {
	if got := plainError(domain.ErrNotGitRepository); !strings.Contains(got, "不是 Git 仓库") {
		t.Fatalf("translation = %q", got)
	}
}

func TestBindAsksForRepositoryAddressWhenMissing(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	model.openDetail()

	// A repository without any remote: gitra must ask for the address instead
	// of silently pushing nowhere.
	dir := filepath.Join(t.TempDir(), "fresh-repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	model.startBind(*model.detailAccount, dir)
	model, _ = press(t, model, "enter") // choose "使用这个文件夹"
	if model.screen != screenRemote {
		t.Fatalf("screen = %v, want remote address step", model.screen)
	}
	if !strings.Contains(model.bind.remoteURL, "github.com/luna/") {
		t.Fatalf("prefilled address should target the account: %q", model.bind.remoteURL)
	}
	view := model.View()
	if !strings.Contains(view, "还没有仓库地址") {
		t.Fatalf("remote step must explain itself:\n%s", view)
	}

	// With an existing origin the flow binds directly.
	repo := newRepoDir(t)
	model.startBind(*model.detailAccount, repo)
	model, _ = press(t, model, "enter")
	if model.screen == screenRemote {
		t.Fatal("existing origin must not trigger the address step")
	}
	if !model.busy {
		t.Fatal("binding should have started")
	}
}
