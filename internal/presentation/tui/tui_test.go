package tui

import (
	"context"
	"fmt"
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
	// Never launch a real browser from tests.
	opened := []string{}
	original := openURL
	openURL = func(url string) error { opened = append(opened, url); return nil }
	t.Cleanup(func() { openURL = original })
	lastOpenedURL = func() string {
		if len(opened) == 0 {
			return ""
		}
		return opened[len(opened)-1]
	}
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
	// First run shows a menu; choosing GitLab must carry the provider over.
	model, _ = press(t, model, "down")
	model, _ = press(t, model, "enter")
	if model.screen != screenLogin || model.login.step != 1 || model.login.providerIndex != 1 {
		t.Fatalf("screen=%v step=%d provider=%d", model.screen, model.login.step, model.login.providerIndex)
	}
	// "A" jumps straight to the login method step (GitHub preselected).
	model, _ = press(t, model, "a")
	if model.screen != screenLogin || model.login.step != 1 {
		t.Fatalf("A shortcut: screen=%v step=%d", model.screen, model.login.step)
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

func TestEmptyStateShowsSelectableMenu(t *testing.T) {
	model, _ := newTestModel(t)
	view := model.View()
	for _, want := range []string{"欢迎使用 gitra", "登录 GitHub", "登录 GitLab", "登录 Gitea", "退出"} {
		if !strings.Contains(view, want) {
			t.Fatalf("welcome menu must show %q:\n%s", want, view)
		}
	}
	model, _ = press(t, model, "enter")
	if model.screen != screenLogin || model.login.step != 1 {
		t.Fatalf("Enter on 登录 GitHub should open login, got screen=%v step=%d", model.screen, model.login.step)
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

func TestLoginRequestNeverReadsStdinInTUI(t *testing.T) {
	request := loginRequest(domain.ProviderGitHub, "", true)
	if request.AllowStdin {
		t.Fatal("the TUI must never let the login path read stdin (Bubble Tea owns the terminal)")
	}
	if !request.AllowCLIReuse {
		t.Fatal("the gh/glab reuse path should stay available")
	}
	withToken := loginRequest(domain.ProviderGitLab, "tok", false)
	if withToken.Token != "tok" || withToken.AllowStdin {
		t.Fatalf("request = %+v", withToken)
	}
}

func TestLoginErrorMessagesAreActionable(t *testing.T) {
	noCLI := loginErrorText(loginDoneMsg{err: fmt.Errorf("%w: no credential", domain.ErrAuthInvalid)})
	if !strings.Contains(noCLI, "粘贴访问码") || strings.Contains(noCLI, "gitra ") {
		t.Fatalf("hint = %q, must guide to the paste option without shell commands", noCLI)
	}
	badToken := loginErrorText(loginDoneMsg{err: fmt.Errorf("%w: rejected", domain.ErrAuthInvalid), usedToken: true})
	if !strings.Contains(badToken, "权限") {
		t.Fatalf("hint = %q, must explain token scopes", badToken)
	}
}

// lastOpenedURL is set by newTestModel; it reports what the UI tried to open.
var lastOpenedURL = func() string { return "" }

// runCmd executes a tea.Cmd (and any follow-up commands) like the runtime does.
func runCmd(t *testing.T, model Model, cmd tea.Cmd) Model {
	t.Helper()
	for i := 0; cmd != nil && i < 10; i++ {
		updated, next := model.Update(cmd())
		model = updated.(Model)
		cmd = next
	}
	return model
}

func clickAt(model Model, line int) Model {
	updated, _ := model.Update(tea.MouseMsg{X: 3, Y: line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	return updated.(Model)
}

func lineOf(model Model, needle string) int {
	lines := strings.Split(model.View(), "\n")
	for index, line := range lines {
		if strings.Contains(line, needle) {
			return index
		}
	}
	return -1
}

func TestMouseClicksWorkOnMenusAndDialogs(t *testing.T) {
	model, _ := newTestModel(t)

	// Click "登录 GitLab" on the welcome menu.
	line := lineOf(model, "登录 GitLab")
	if line < 0 {
		t.Fatal("welcome menu line not found")
	}
	model = clickAt(model, line)
	if model.screen != screenLogin || model.login.step != 1 || model.login.providerIndex != 1 {
		t.Fatalf("click did not start GitLab login: screen=%v step=%d provider=%d",
			model.screen, model.login.step, model.login.providerIndex)
	}

	// Discovery finishes: the list shows the manual fallback plus any found login.
	updated, _ := model.Update(candidatesMsg{})
	model = updated.(Model)

	// Click "粘贴访问码" to reach the token input.
	line = lineOf(model, "粘贴访问码")
	if line < 0 {
		t.Fatal("method line not found")
	}
	model, cmd := func() (Model, tea.Cmd) {
		updated, c := model.Update(tea.MouseMsg{X: 3, Y: line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		return updated.(Model), c
	}()
	model = runCmd(t, model, cmd)
	if model.login.step != 2 {
		t.Fatalf("click did not open the token input: step=%d", model.login.step)
	}
	// This test chose GitLab above, so the GitLab token page must open.
	if got := lastOpenedURL(); !strings.Contains(got, "personal_access_tokens") || !strings.Contains(got, "scopes=") {
		t.Fatalf("token page should be opened with scopes prefilled, got %q", got)
	}

	// Click through the confirm dialog.
	ran := false
	model.screen = screenConfirm
	model.confirmPrompt = "测试"
	model.confirmAction = func() tea.Cmd { ran = true; return nil }
	line = lineOf(model, "[Y] 确认")
	if line < 0 {
		t.Fatal("confirm button not found")
	}
	model = clickAt(model, line)
	if !ran {
		t.Fatal("clicking [Y] 确认 must run the action")
	}
}

func TestMouseClickOpensAccountAndAddButton(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()

	line := lineOf(model, "luna")
	if line < 0 {
		t.Fatal("account card not found")
	}
	model = clickAt(model, line)
	if model.screen != screenDetail || model.detailAccount == nil {
		t.Fatalf("clicking a card must open the detail, got %v", model.screen)
	}

	model, _ = press(t, model, "esc")
	line = lineOf(model, "[+ 添加账号]")
	if line < 0 {
		t.Fatal("add-account entry not found")
	}
	model = clickAt(model, line)
	if model.screen != screenLogin {
		t.Fatalf("clicking [+ 添加账号] must open login, got %v", model.screen)
	}
}

func TestStripANSIRemovesStyling(t *testing.T) {
	styled := accentStyle.Render("登录 GitHub")
	if got := stripANSI(styled); got != "登录 GitHub" {
		t.Fatalf("stripANSI = %q", got)
	}
}

func TestDiscoveredLoginsAreOfferedFirst(t *testing.T) {
	model, _ := newTestModel(t)

	// Opening login starts discovery.
	model, cmd := press(t, model, "a")
	if !model.login.detecting {
		t.Fatal("login should start by detecting reusable logins")
	}
	if cmd == nil {
		t.Fatal("detection must run as a command")
	}
	if !strings.Contains(model.View(), "正在检查这台电脑上已有的登录方式") {
		t.Fatalf("view must show progress:\n%s", model.View())
	}

	// Discovery result: one verified SSH key plus the manual fallback.
	updated, _ := model.Update(candidatesMsg{candidates: []app.Candidate{{
		Kind: "ssh-key", Source: "ssh", Provider: domain.ProviderGitHub, Host: "github.com",
		Username: "lunafoundry", KeyPath: "/Users/x/.ssh/id_ed25519_luna",
		Label: "使用已有密钥 id_ed25519_luna（已验证属于 lunafoundry）",
	}}})
	model = updated.(Model)
	view := model.View()
	if !strings.Contains(view, "使用已有密钥 id_ed25519_luna") || !strings.Contains(view, "粘贴访问码") {
		t.Fatalf("options must show the discovered key first:\n%s", view)
	}

	// Selecting the discovered key starts that login (no typing at all).
	model, cmd = press(t, model, "enter")
	if !model.busy || cmd == nil || !strings.Contains(model.message, "id_ed25519_luna") {
		t.Fatalf("selecting an ssh candidate should start login: busy=%v msg=%q", model.busy, model.message)
	}
}

func TestManualOptionOpensTokenPageWithScopes(t *testing.T) {
	model, _ := newTestModel(t)
	model, _ = press(t, model, "a")
	updated, _ := model.Update(candidatesMsg{})
	model = updated.(Model)

	model, cmd := press(t, model, "enter") // only option: manual
	if model.login.step != 2 || cmd == nil {
		t.Fatalf("manual option must open the token step: step=%d", model.login.step)
	}
	model = runCmd(t, model, cmd)
	if got := lastOpenedURL(); !strings.Contains(got, "scopes=repo,read:user,user:email") {
		t.Fatalf("token page must prefill scopes, got %q", got)
	}
	view := model.View()
	if !strings.Contains(view, "权限：repo、read:user、user:email") {
		t.Fatalf("manual step must spell out the scopes:\n%s", view)
	}
}
