package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/domain"
)

func (m *Model) ctx() context.Context { return context.Background() }

// loginRequest builds the request for one login attempt.
//
// AllowStdin is always false in the TUI: the terminal is owned by Bubble Tea,
// so reading stdin here would block the interface. The access code comes from
// the masked input field instead.
func loginRequest(provider domain.ProviderType, token string, allowCLI bool) app.LoginRequest {
	return app.LoginRequest{
		Provider:      provider,
		Token:         token,
		AllowStdin:    false,
		AllowCLIReuse: allowCLI,
	}
}

// loginCommand runs the provider login (explicit user action only).
func (m *Model) loginCommand(provider domain.ProviderType, token string, allowCLI bool) tea.Cmd {
	login := m.app.Login
	return func() tea.Msg {
		if login == nil {
			return loginDoneMsg{err: fmt.Errorf("登录功能不可用")}
		}
		result, err := login.Login(context.Background(), loginRequest(provider, token, allowCLI))
		if err != nil {
			return loginDoneMsg{err: err, usedToken: token != ""}
		}
		return loginDoneMsg{
			alias: result.Account.Alias, username: result.Profile.Username,
			host: result.Account.Provider.Endpoint.Host, created: result.Created,
		}
	}
}

func (m *Model) bindCommand(account domain.Account, path string) tea.Cmd {
	bindings := m.app.Bindings
	return func() tea.Msg {
		if _, err := bindings.Bind(context.Background(), app.BindRequest{AccountID: account.ID, Path: path}); err != nil {
			return bindDoneMsg{path: path, alias: account.Alias, err: err}
		}
		return bindDoneMsg{path: path, alias: account.Alias}
	}
}

func (m *Model) unbindCommand(path string) tea.Cmd {
	bindings := m.app.Bindings
	return func() tea.Msg {
		if err := bindings.Unbind(context.Background(), app.UnbindRequest{Path: path}); err != nil {
			return unbindDoneMsg{path: path, err: err}
		}
		return unbindDoneMsg{path: path}
	}
}

func (m *Model) verifyCommand(account domain.Account) tea.Cmd {
	verification := m.app.Verification
	return func() tea.Msg {
		if verification == nil {
			return verifyDoneMsg{message: "验证功能不可用"}
		}
		result, err := verification.VerifyAccount(context.Background(), account.ID)
		if err != nil {
			return verifyDoneMsg{message: err.Error()}
		}
		return verifyDoneMsg{success: result.Success, status: string(result.Status), message: result.Message}
	}
}

func (m *Model) removeAccountCommand(account domain.Account) tea.Cmd {
	accounts := m.app.Accounts
	return func() tea.Msg {
		if err := accounts.Delete(context.Background(), app.DeleteAccountRequest{ID: account.ID, UnbindAll: true}); err != nil {
			return accountRemovedMsg{alias: account.Alias, err: err}
		}
		return accountRemovedMsg{alias: account.Alias}
	}
}

func (m *Model) reconcileCommand(account domain.Account) tea.Cmd {
	reconciler := m.app.Reconciler
	return func() tea.Msg {
		result, err := reconciler.ReconcileAccount(context.Background(), account.ID)
		if err != nil {
			return bindDoneMsg{err: err}
		}
		if len(result.Failures) > 0 {
			return bindDoneMsg{err: fmt.Errorf("%v", result.Failures)}
		}
		return bindDoneMsg{}
	}
}

// loadAccounts refreshes the visible data from local stores (no network).
func (m *Model) loadAccounts() {
	accounts, err := m.app.Accounts.List(m.ctx())
	if err != nil {
		m.errText = plainError(err)
		return
	}
	cards := make([]AccountCard, 0, len(accounts))
	for _, account := range accounts {
		bindings, err := m.app.Deps.Bindings.ListByAccount(m.ctx(), account.ID)
		if err != nil {
			m.errText = plainError(err)
			return
		}
		cards = append(cards, m.cardFor(account, len(bindings)))
	}
	m.accounts = cards
	if m.selected >= len(cards) {
		m.selected = max(0, len(cards)-1)
	}
}

func (m *Model) accountByID(id string) (domain.Account, bool) {
	accounts, err := m.app.Accounts.List(m.ctx())
	if err != nil {
		return domain.Account{}, false
	}
	for _, account := range accounts {
		if string(account.ID) == id {
			return account, true
		}
	}
	return domain.Account{}, false
}

func (m *Model) cardFor(account domain.Account, projects int) AccountCard {
	state := "configured"
	if strategy, err := m.app.Deps.Auth.Get(account.Transport.Strategy); err != nil {
		state = "invalid"
	} else if err := strategy.Validate(m.ctx(), account); err != nil {
		if account.Transport.Strategy == domain.StrategyHTTPSToken {
			state = "needs_login"
		} else {
			state = "invalid"
		}
	}
	return AccountCard{
		ID:           string(account.ID),
		Provider:     providerLabel(account.Provider.Type),
		Alias:        account.Alias,
		Host:         account.Provider.Endpoint.Host,
		AuthLabel:    authLabel(account),
		LocalState:   state,
		ProjectCount: projects,
	}
}

func (m *Model) openDetail() {
	if m.selected < 0 || m.selected >= len(m.accounts) {
		return
	}
	card := m.accounts[m.selected]
	account, ok := m.accountByID(card.ID)
	if !ok {
		m.errText = "账号数据已变化，请重试"
		return
	}
	m.detailAccount = &account
	m.detailCard = card
	m.detailSelected = 0
	m.loadDetailProjects(account)
	m.screen = screenDetail
}

func (m *Model) loadDetailProjects(account domain.Account) {
	bindings, err := m.app.Deps.Bindings.ListByAccount(m.ctx(), account.ID)
	if err != nil {
		m.errText = plainError(err)
		return
	}
	projects := make([]Project, 0, len(bindings))
	for _, binding := range bindings {
		state := "ok"
		if status, err := m.app.Bindings.Status(m.ctx(), binding.Repository.Path); err == nil {
			state = string(status.Health)
			if status.NeedsLogin {
				state = "needs_login"
			}
		} else {
			state = "missing"
		}
		projects = append(projects, Project{ID: string(binding.ID), Alias: account.Alias, Path: binding.Repository.Path, State: state})
	}
	m.detailProjects = projects
	if m.detailSelected >= len(projects) {
		m.detailSelected = max(0, len(projects)-1)
	}
}

// startBind opens the folder picker rooted at the current directory.
func (m *Model) startBind(account domain.Account, path string) {
	entries, err := listDirs(path)
	if err != nil {
		m.errText = "无法读取文件夹：" + err.Error()
		return
	}
	m.bind = bindState{path: path, entries: entries, account: account}
	m.screen = screenBind
}

// listDirs lists the selectable folders inside path.
//
// Entries are resolved with os.Stat so symlinked folders are included: on macOS
// with iCloud "Desktop & Documents" sync enabled, ~/Desktop and ~/Documents are
// symlinks, and a plain DirEntry.IsDir() would silently hide them.
func listDirs(path string) ([]string, error) {
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, item := range items {
		name := item.Name()
		if strings.HasPrefix(name, ".") {
			continue // keep caches and config folders out of the way
		}
		if item.IsDir() {
			dirs = append(dirs, name)
			continue
		}
		if info, err := os.Stat(filepath.Join(path, name)); err == nil && info.IsDir() {
			dirs = append(dirs, name) // symlink to a folder
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// ensureAndCheckRemote stores a missing origin remote, then asks the provider
// whether the repository really exists. This is what prevents the "bound to a
// repository that does not exist" trap.
func (m *Model) ensureAndCheckRemoteCommand(account domain.Account, path, url string) tea.Cmd {
	bindings := m.app.Bindings
	remote := m.app.Remote
	return func() tea.Msg {
		ctx := context.Background()
		if strings.TrimSpace(url) != "" {
			if err := bindings.EnsureOriginRemote(ctx, path, url); err != nil {
				return remoteCheckMsg{err: err}
			}
		}
		if remote == nil {
			return remoteCheckMsg{err: fmt.Errorf("在线校验不可用")}
		}
		check, err := remote.Check(ctx, account.ID, path)
		return remoteCheckMsg{check: check, err: err}
	}
}

// createAndBindCommand creates the provider-side repository and binds.
func (m *Model) createAndBindCommand(account domain.Account, path string) tea.Cmd {
	remote := m.app.Remote
	return func() tea.Msg {
		if remote == nil {
			return bindDoneMsg{path: path, alias: account.Alias, err: fmt.Errorf("在线创建不可用")}
		}
		if _, err := remote.CreateAndBind(context.Background(), account.ID, path, true); err != nil {
			return bindDoneMsg{path: path, alias: account.Alias, err: err}
		}
		return bindDoneMsg{path: path, alias: account.Alias}
	}
}

// bindOnlyCommand binds a folder whose remote is already confirmed.
func (m *Model) bindOnlyCommand(account domain.Account, path string) tea.Cmd {
	bindings := m.app.Bindings
	return func() tea.Msg {
		if _, err := bindings.Bind(context.Background(), app.BindRequest{AccountID: account.ID, Path: path}); err != nil {
			return bindDoneMsg{path: path, alias: account.Alias, err: err}
		}
		return bindDoneMsg{path: path, alias: account.Alias}
	}
}

// lookPath is replaceable so tests can simulate an installed CLI.
var lookPath = exec.LookPath

// browserLoginCLI reports which official CLI can perform a browser login.
func browserLoginCLI(providerType domain.ProviderType) (string, bool) {
	var binary string
	switch providerType {
	case domain.ProviderGitHub:
		binary = "gh"
	case domain.ProviderGitLab:
		binary = "glab"
	default:
		return "", false
	}
	path, err := lookPath(binary)
	if err != nil {
		return "", false
	}
	return path, true
}

// browserLoginCommand hands the terminal to the official CLI (gh/glab) so the
// user authorizes in the browser; gitra resumes and re-reads the session.
func (m *Model) browserLoginCommand(providerType domain.ProviderType, host string) tea.Cmd {
	path, ok := browserLoginCLI(providerType)
	if !ok {
		return func() tea.Msg {
			return cliLoginResultMsg{err: fmt.Errorf("没有找到官方客户端")}
		}
	}
	var args []string
	if providerType == domain.ProviderGitHub {
		args = []string{"auth", "login", "--hostname", host, "--git-protocol", "https", "--web"}
	} else {
		args = []string{"auth", "login", "--hostname", host, "--web"}
	}
	command := exec.Command(path, args...)
	return tea.ExecProcess(command, func(err error) tea.Msg {
		return cliLoginResultMsg{err: err}
	})
}

// detectCommand looks for reusable logins already present on this machine.
func (m *Model) detectCommand(provider domain.ProviderType, host string) tea.Cmd {
	detector := m.app.Detector
	return func() tea.Msg {
		if detector == nil {
			return candidatesMsg{}
		}
		return candidatesMsg{candidates: detector.Detect(context.Background(), provider, host)}
	}
}

// sshKeyLoginCommand registers an account that uses a key the provider
// already accepts.
func (m *Model) sshKeyLoginCommand(candidate app.Candidate) tea.Cmd {
	login := m.app.Login
	return func() tea.Msg {
		if login == nil {
			return loginDoneMsg{err: fmt.Errorf("登录功能不可用")}
		}
		result, err := login.LoginWithSSHKey(context.Background(), app.SSHLoginRequest{
			Provider: providerForSSH(candidate), Host: candidate.Host,
			KeyPath: candidate.KeyPath, Username: candidate.Username,
		})
		if err != nil {
			return loginDoneMsg{err: err}
		}
		return loginDoneMsg{alias: result.Account.Alias, username: result.Profile.Username, host: candidate.Host, created: result.Created}
	}
}

func providerForSSH(candidate app.Candidate) domain.ProviderType {
	if candidate.Provider != "" {
		return candidate.Provider
	}
	return domain.ProviderGitHub
}
