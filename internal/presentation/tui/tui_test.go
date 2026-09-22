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
	withFakeLookPath(t, map[string]string{})
	configDir := t.TempDir()
	// Never touch the real system keychain from tests.
	t.Setenv("GITRA_CREDENTIAL_HELPER", "store --file="+filepath.Join(configDir, "credentials"))
	t.Setenv("GITRA_CONFIG_DIR", configDir)
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

func gitRunT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
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

// dragFromTo simulates dragging (for example to select text): press somewhere,

func lineOf(model Model, needle string) int {
	lines := strings.Split(model.View(), "\n")
	for index, line := range lines {
		if strings.Contains(line, needle) {
			return index
		}
	}
	return -1
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

	model.login.methodIndex = len(model.loginOptions()) - 1 // the manual row
	model, cmd := press(t, model, "enter")
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

func TestQuitAlwaysAsksForConfirmation(t *testing.T) {
	model, _ := newTestModel(t)

	// "q" on the accounts screen asks first.
	model, _ = press(t, model, "q")
	if model.quit || model.screen != screenConfirm {
		t.Fatalf("q must ask for confirmation first: screen=%v quit=%v", model.screen, model.quit)
	}
	if !strings.Contains(model.confirmPrompt, "要退出 gitra 吗") {
		t.Fatalf("prompt = %q", model.confirmPrompt)
	}

	// Cancelling returns to the menu without quitting.
	model, _ = press(t, model, "n")
	if model.quit || model.screen != screenAccounts {
		t.Fatalf("cancel must return to the menu: screen=%v quit=%v", model.screen, model.quit)
	}

	// Confirming quits.
	model, _ = press(t, model, "q")
	model, cmd := press(t, model, "y")
	model = runCmd(t, model, cmd)
	if !model.quit {
		t.Fatal("confirming must quit")
	}
}

func TestKeyboardNavigationOnly(t *testing.T) {
	model, _ := newTestModel(t)

	// Welcome menu: ↓ moves, Enter opens the chosen provider.
	model, _ = press(t, model, "down")
	model, _ = press(t, model, "enter")
	if model.screen != screenLogin || model.login.providerIndex != 1 {
		t.Fatalf("keyboard selection failed: screen=%v provider=%d", model.screen, model.login.providerIndex)
	}

	// Esc backs out step by step.
	model, _ = press(t, model, "esc")
	model, _ = press(t, model, "esc")
	if model.screen != screenAccounts {
		t.Fatalf("esc must return to the accounts screen, got %v", model.screen)
	}
}

func TestAccountCardOpensWithEnter(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()

	model, _ = press(t, model, "enter")
	if model.screen != screenDetail || model.detailAccount == nil {
		t.Fatalf("Enter must open the account detail, got %v", model.screen)
	}
	model, _ = press(t, model, "esc")
	if model.screen != screenAccounts {
		t.Fatalf("Esc must return to accounts, got %v", model.screen)
	}
}

func TestTimeoutErrorsAreReadable(t *testing.T) {
	got := plainError(context.DeadlineExceeded)
	if !strings.Contains(got, "超时") || strings.Contains(got, "context deadline") {
		t.Fatalf("timeout translation = %q", got)
	}
	if got := plainError(context.Canceled); !strings.Contains(got, "超时") {
		t.Fatalf("cancel translation = %q", got)
	}
}

func TestEscCancelsWaitingState(t *testing.T) {
	model, _ := newTestModel(t)
	model.busy = true
	model.message = "正在登录…"

	model, _ = press(t, model, "esc")
	if model.busy {
		t.Fatal("Esc must clear the waiting state")
	}
	if !strings.Contains(model.errText, "已停止等待") {
		t.Fatalf("notice = %q", model.errText)
	}

	// Any other key is still ignored while waiting.
	model.busy = true
	model.errText = ""
	model, _ = press(t, model, "a")
	if !model.busy || model.errText != "" {
		t.Fatalf("keys other than Esc must be ignored while busy: busy=%v err=%q", model.busy, model.errText)
	}
}

func withFakeLookPath(t *testing.T, available map[string]string) {
	t.Helper()
	original := lookPath
	lookPath = func(binary string) (string, error) {
		if path, ok := available[binary]; ok {
			return path, nil
		}
		return "", fmt.Errorf("%s not found", binary)
	}
	t.Cleanup(func() { lookPath = original })
}

func TestBrowserLoginOptionAppearsWhenCLIExists(t *testing.T) {
	model, _ := newTestModel(t)
	withFakeLookPath(t, map[string]string{"gh": "/opt/homebrew/bin/gh"})
	model, _ = press(t, model, "a")
	updated, _ := model.Update(candidatesMsg{})
	model = updated.(Model)

	view := model.View()
	if !strings.Contains(view, "在浏览器里登录 GitHub") {
		t.Fatalf("browser login option missing:\n%s", view)
	}
	if !strings.Contains(view, "不需要创建或粘贴任何内容") {
		t.Fatalf("hint must explain the frictionless option:\n%s", view)
	}

	// Choosing it suspends the TUI and starts the official CLI login.
	model, _ = press(t, model, "down") // past "粘贴访问码" is index order dependent; use first option
	model.login.methodIndex = 0
	model, cmd := press(t, model, "enter")
	if !model.busy || cmd == nil {
		t.Fatalf("browser login must start: busy=%v", model.busy)
	}
	if !strings.Contains(model.message, "浏览器") {
		t.Fatalf("message = %q", model.message)
	}
}

func TestBrowserLoginOptionHiddenWithoutCLI(t *testing.T) {
	model, _ := newTestModel(t)
	withFakeLookPath(t, map[string]string{})
	model, _ = press(t, model, "a")
	updated, _ := model.Update(candidatesMsg{})
	model = updated.(Model)
	if strings.Contains(model.View(), "在浏览器里登录") {
		t.Fatalf("no CLI installed: the browser option must be hidden:\n%s", model.View())
	}
}

func TestBrowserLoginResumesIntoCLISession(t *testing.T) {
	model, _ := newTestModel(t)
	withFakeLookPath(t, map[string]string{"gh": "/opt/homebrew/bin/gh"})
	model, _ = press(t, model, "a")
	updated, _ := model.Update(candidatesMsg{})
	model = updated.(Model)

	// The user finished the browser flow.
	updated, cmd := model.Update(cliLoginResultMsg{})
	model = updated.(Model)
	if !model.login.detecting || !model.login.autoPickCLI || cmd == nil {
		t.Fatalf("after authorization gitra must re-read the session: detecting=%v autoPick=%v",
			model.login.detecting, model.login.autoPickCLI)
	}

	// The freshly authorized CLI session is picked up automatically.
	updated, cmd = model.Update(candidatesMsg{candidates: []app.Candidate{{
		Kind: "cli", Source: "gh", Provider: domain.ProviderGitHub, Host: "github.com",
		Label: "使用本机已登录的 GitHub（无需输入）",
	}}})
	model = updated.(Model)
	if !model.busy || !strings.Contains(model.message, "本机已登录") {
		t.Fatalf("the authorized session must be used automatically: busy=%v msg=%q", model.busy, model.message)
	}
	if cmd == nil {
		t.Fatal("expected a login command")
	}
}

func TestLoginOffersToAddProjectAnywhere(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	// Launch from a folder that is not a repository (like the app bundle does).
	t.Chdir(t.TempDir())
	model.loadAccounts()

	updated, _ := model.Update(loginDoneMsg{alias: "luna", username: "labuno", host: "github.com", created: false})
	model = updated.(Model)
	if model.screen != screenConfirm {
		t.Fatalf("after login the user must be asked about the first project, screen=%v", model.screen)
	}
	if !strings.Contains(model.confirmPrompt, "要现在添加一个项目吗") {
		t.Fatalf("prompt = %q", model.confirmPrompt)
	}

	// Confirming opens the folder picker for that account.
	model, cmd := press(t, model, "y")
	model = runCmd(t, model, cmd)
	if model.screen != screenBind {
		t.Fatalf("confirming must open the folder picker, screen=%v", model.screen)
	}
	if model.bind.account.Alias != "luna" {
		t.Fatalf("picker bound to %q, want luna", model.bind.account.Alias)
	}
}

func TestBindKeyOnAccountsScreen(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()

	model, _ = press(t, model, "b")
	if model.screen != screenBind {
		t.Fatalf("B must open the folder picker for the selected account, screen=%v", model.screen)
	}
	if model.bind.path == "" {
		t.Fatal("picker must start in a real folder")
	}
}

func TestAccountsScreenHasNoDecorativeAddRow(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	if strings.Contains(model.View(), "[+ 添加账号]") {
		t.Fatalf("the decorative add row must be gone (A is the documented entry):\n%s", model.View())
	}
	if !strings.Contains(model.View(), "B 绑定项目") {
		t.Fatalf("help line must mention B:\n%s", model.View())
	}
}

func TestWindowRangeKeepsSelectionVisible(t *testing.T) {
	tests := []struct {
		name             string
		total, sel, size int
		wantStart        int
		wantEnd          int
	}{
		{"everything fits", 3, 2, 10, 0, 3},
		{"top of a long list", 40, 0, 10, 0, 10},
		{"middle", 40, 20, 10, 15, 25},
		{"bottom", 40, 39, 10, 30, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := windowRange(tt.total, tt.sel, tt.size)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("windowRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.total, tt.sel, tt.size, start, end, tt.wantStart, tt.wantEnd)
			}
			if tt.sel < start || tt.sel >= end {
				t.Fatalf("selection %d not visible in [%d, %d)", tt.sel, start, end)
			}
		})
	}
}

func TestBindPickerScrollsToSelection(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	model.openDetail()

	// A folder with more subfolders than fit on a small screen.
	root := t.TempDir()
	for index := 0; index < 40; index++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("dir-%02d", index)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	model.height = 20
	model.startBind(*model.detailAccount, root)
	if len(model.bind.entries) != 40 {
		t.Fatalf("entries = %d, want 40", len(model.bind.entries))
	}

	// Walk to the bottom: the view must follow the cursor and say how many
	// entries remain above.
	for index := 0; index < 41; index++ {
		model, _ = press(t, model, "down")
	}
	view := model.View()
	if !strings.Contains(view, "> ") {
		t.Fatalf("missing selection marker:\n%s", view)
	}
	if !strings.Contains(view, "上面还有") {
		t.Fatalf("the view must indicate hidden entries above:\n%s", view)
	}
	if !strings.Contains(view, model.bind.entries[len(model.bind.entries)-1]) {
		t.Fatalf("the last entry must be reachable and visible:\n%s", view)
	}
}

func TestBindViewExplainsEnter(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	model.openDetail()
	model.startBind(*model.detailAccount, t.TempDir())

	view := model.View()
	if !strings.Contains(view, "回车 = 绑定当前文件夹") {
		t.Fatalf("the picker must explain what Enter does:\n%s", view)
	}
	if !strings.Contains(view, "使用这个文件夹（回车＝绑定它）") {
		t.Fatalf("the bind row must be labelled:\n%s", view)
	}
	if !strings.Contains(model.helpLine(), "在「使用这个文件夹」上＝绑定") {
		t.Fatalf("help line = %q", model.helpLine())
	}
}

func TestListDirsIncludesSymlinkedFolders(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "Documents")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, "LinkedDocs")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dirs, err := listDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, dir := range dirs {
		found[dir] = true
	}
	if !found["Documents"] || !found["LinkedDocs"] {
		t.Fatalf("listDirs = %v, want both real and symlinked folders", dirs)
	}
}

func TestHelpNeverExceedsWindowWidth(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	model.openDetail()

	for _, width := range []int{60, 80, 100, 120, 200} {
		model.width = width
		rendered := model.renderHelp()
		for _, line := range strings.Split(rendered, "\n") {
			if got := displayWidth(line); got > width {
				t.Fatalf("width %d: help line %q is %d cells wide", width, line, got)
			}
		}
		// Every hint must survive: wrapping is allowed, losing text is not.
		for _, segment := range model.helpSegments() {
			if !strings.Contains(rendered, segment) {
				t.Fatalf("width %d: hint %q missing from:\n%s", width, segment, rendered)
			}
		}
	}
}

func TestHelpWrapsInsteadOfClipping(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	model.openDetail()
	model.width = 60

	rendered := model.renderHelp()
	if !strings.Contains(rendered, "\n") {
		t.Fatalf("a narrow window must wrap the help:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Esc 返回") {
		t.Fatalf("the last hint must still be visible:\n%s", rendered)
	}
}

func TestProjectRowsUseReadableState(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	model.openDetail()
	model.detailProjects = []Project{{ID: "bnd_x", Alias: "luna", Path: "/tmp/site", State: "drift"}}
	view := model.View()
	if !strings.Contains(view, "（drift）") || !strings.Contains(view, "● 配置被改动") {
		t.Fatalf("project row must show label and raw state:\n%s", view)
	}
}

func TestRemoteAddressIsVerifiedBeforeBinding(t *testing.T) {
	model, application := newTestModel(t)
	addSSHAccount(t, application, "luna")
	model.loadAccounts()
	account := mustAccount(t, application, "luna")

	// The folder picker stored an address; the user confirms it with Enter.
	model.screen = screenRemote
	model.bind = bindState{path: t.TempDir(), account: account}
	model.bind.remoteURL = "https://github.com/labuno/gitra.git"

	model, cmd := press(t, model, "enter")
	if !model.busy || cmd == nil || !strings.Contains(model.message, "校验") {
		t.Fatalf("Enter must verify the address first: busy=%v msg=%q", model.busy, model.message)
	}

	// The provider says the repository exists: bind directly.
	updated, cmd := model.Update(remoteCheckMsg{check: app.RemoteCheck{Exists: true, Owner: "labuno", Name: "gitra"}})
	model = updated.(Model)
	if !model.busy || cmd == nil || !strings.Contains(model.message, "已确认仓库存在") {
		t.Fatalf("existing repository must proceed to bind: busy=%v msg=%q", model.busy, model.message)
	}

	// The provider says it does not exist: ask before creating anything.
	updated, _ = model.Update(remoteCheckMsg{check: app.RemoteCheck{Exists: false, Owner: "labuno", Name: "gitra"}})
	model = updated.(Model)
	if model.screen != screenConfirm {
		t.Fatalf("missing repository must ask for confirmation, screen=%v", model.screen)
	}
	if !strings.Contains(model.confirmPrompt, "要现在创建吗") || !strings.Contains(model.confirmPrompt, "labuno/gitra") {
		t.Fatalf("prompt = %q", model.confirmPrompt)
	}

	// Confirming runs the creation flow and reports an outcome.
	model, cmd = press(t, model, "y")
	if cmd == nil {
		t.Fatal("confirming must start the create-and-bind command")
	}
	model = runCmd(t, model, cmd)
	if model.errText == "" && model.message == "" {
		t.Fatal("the create flow must report an outcome")
	}
}

func mustAccount(t *testing.T, application *bootstrap.App, alias string) domain.Account {
	t.Helper()
	account, err := application.Accounts.GetByAlias(context.Background(), alias)
	if err != nil {
		t.Fatal(err)
	}
	return account
}

// addHTTPSAccount registers an account that uses a stored credential, so HTTPS
// remotes can be bound (the ssh-key path is covered by addSSHAccount).
func addHTTPSAccount(t *testing.T, application *bootstrap.App, alias string) domain.Account {
	t.Helper()
	ctx := context.Background()
	ref := "github.com/" + alias
	if err := application.Deps.Secrets.Set(ctx, ref, "test-token"); err != nil {
		t.Fatalf("store credential: %v", err)
	}
	account, err := application.Accounts.Create(ctx, app.CreateAccountRequest{
		Alias: alias,
		Provider: domain.ProviderRef{
			Type: domain.ProviderGitHub, Username: alias,
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: alias, Email: alias + "@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategyHTTPSToken, Config: map[string]string{}},
	})
	if err != nil {
		t.Fatalf("create https account: %v", err)
	}
	return account
}

// makeRepoReadyToPublish creates a repository with one commit and a local bare
// remote, so the publish flow can be exercised offline.
func makeRepoReadyToPublish(t *testing.T) (repo string, bare string) {
	t.Helper()
	base := t.TempDir()
	bare = filepath.Join(base, "remote.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRunT(t, filepath.Dir(bare), "init", "-q", "--bare", bare)

	repo = filepath.Join(base, "site")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRunT(t, repo, "init", "-q", "-b", "main")
	gitRunT(t, repo, "config", "user.name", "Luna")
	gitRunT(t, repo, "config", "user.email", "luna@example.com")
	gitRunT(t, repo, "remote", "add", "origin", "https://github.com/lunafoundry/site.git")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRunT(t, repo, "add", ".")
	gitRunT(t, repo, "commit", "-q", "-m", "initial")
	return repo, bare
}

func TestPublishKeyUploadsFirstTime(t *testing.T) {
	model, application := newTestModel(t)
	model.loadAccounts()
	account := addHTTPSAccount(t, application, "luna")

	repo, bare := makeRepoReadyToPublish(t)
	ctx := context.Background()
	if _, err := application.Bindings.Bind(ctx, app.BindRequest{AccountID: account.ID, Path: repo}); err != nil {
		t.Fatalf("bind: %v", err)
	}
	// Binding validated the real remote; point origin at the local bare repo so
	// the publish test stays offline.
	gitRunT(t, repo, "remote", "set-url", "origin", bare)

	model.loadAccounts()
	model.openDetail()
	model, cmd := press(t, model, "u")
	if !model.busy || cmd == nil {
		t.Fatalf("U must start the first upload: busy=%v", model.busy)
	}
	model = runCmd(t, model, cmd)
	if !strings.Contains(model.message, "已上传") {
		t.Fatalf("message = %q, err = %q", model.message, model.errText)
	}
	if out := gitRunT(t, bare, "branch", "--list"); !strings.Contains(out, "main") {
		t.Fatalf("remote branches = %q, want main", out)
	}
	if !strings.Contains(model.helpLine(), "U 首次上传") && !strings.Contains(model.renderHelp(), "U 首次上传") {
		t.Fatalf("help must mention the upload key: %q", model.renderHelp())
	}
}
