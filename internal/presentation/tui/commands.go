package tui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/domain"
)

func (m *Model) ctx() context.Context { return context.Background() }

// loginCommand runs the provider login (explicit user action only).
func (m *Model) loginCommand(provider domain.ProviderType, token string, allowCLI bool) tea.Cmd {
	login := m.app.Login
	return func() tea.Msg {
		if login == nil {
			return loginDoneMsg{err: fmt.Errorf("登录功能不可用")}
		}
		result, err := login.Login(context.Background(), app.LoginRequest{
			Provider: provider, Token: token, AllowStdin: token == "", AllowCLIReuse: allowCLI,
		})
		if err != nil {
			return loginDoneMsg{err: err}
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

func listDirs(path string) ([]string, error) {
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, item := range items {
		if item.IsDir() && !strings.HasPrefix(item.Name(), ".") {
			dirs = append(dirs, item.Name())
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}
