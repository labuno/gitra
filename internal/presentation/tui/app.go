package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
)

// Run starts the interactive TUI.
func Run(application *bootstrap.App) error {
	program := tea.NewProgram(New(application), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// Init implements tea.Model. Startup performs local reads only.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quit = true
			return m, tea.Quit
		}
		if m.busy {
			return m, nil
		}
		return m.handleKey(msg)

	case loginDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		m.errText = ""
		action := "已登录"
		if !msg.created {
			action = "已更新登录"
		}
		m.message = action + "：" + msg.username + "（" + msg.host + "）"
		m.screen = screenAccounts
		m.loadAccounts()
		return m, m.offerOnboardingBind(msg.alias)

	case bindDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		if msg.path != "" {
			m.message = "已绑定：" + msg.path + " → " + msg.alias
			m.errText = ""
		} else {
			m.message = "配置已检查并修复"
			m.errText = ""
		}
		if m.detailAccount != nil {
			m.loadDetailProjects(*m.detailAccount)
		}
		m.loadAccounts()
		m.screen = screenDetail
		if m.detailAccount == nil {
			m.screen = screenAccounts
		}
		return m, nil

	case unbindDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		m.message = "已解除绑定：" + msg.path
		m.errText = ""
		if m.detailAccount != nil {
			m.loadDetailProjects(*m.detailAccount)
		}
		m.loadAccounts()
		return m, nil

	case verifyDoneMsg:
		m.busy = false
		if msg.success {
			m.message = "连接正常：" + msg.message
			m.errText = ""
		} else {
			m.message = ""
			m.errText = "连接检查未通过：" + msg.message
		}
		return m, nil

	case accountRemovedMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		m.message = "已删除账号：" + msg.alias
		m.errText = ""
		m.detailAccount = nil
		m.screen = screenAccounts
		m.loadAccounts()
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenAccounts:
		return m.handleAccountsKey(msg)
	case screenDetail:
		return m.handleDetailKey(msg)
	case screenLogin:
		return m.handleLoginKey(msg)
	case screenBind:
		return m.handleBindKey(msg)
	case screenConfirm:
		return m.handleConfirmKey(msg)
	}
	return m, nil
}

func (m Model) handleAccountsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	columns := m.columns()
	switch msg.String() {
	case "q":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if m.selected-columns >= 0 {
			m.selected -= columns
		}
	case "down", "j":
		if m.selected+columns < len(m.accounts) {
			m.selected += columns
		}
	case "left", "h":
		if m.selected > 0 {
			m.selected--
		}
	case "right", "l":
		if m.selected < len(m.accounts)-1 {
			m.selected++
		}
	case "enter":
		if len(m.accounts) == 0 {
			m.screen = screenLogin
			m.login = loginState{}
			return m, nil
		}
		m.message, m.errText = "", ""
		m.openDetail()
		if m.detailAccount != nil {
			m.screen = screenDetail
		}
	case "a":
		m.message, m.errText = "", ""
		m.login = loginState{}
		m.screen = screenLogin
	case "t":
		if len(m.accounts) == 0 {
			m.errText = "还没有账号，先按 A 登录一个。"
			return m, nil
		}
		if account, ok := m.accountByID(m.accounts[m.selected].ID); ok {
			m.message = "正在检查 " + account.Alias + " …"
			m.busy = true
			return m, m.verifyCommand(account)
		}
	case "esc":
		m.message, m.errText = "", ""
	}
	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	account := m.detailAccount
	switch msg.String() {
	case "esc":
		m.screen = screenAccounts
		m.message, m.errText = "", ""
		return m, nil
	case "up", "k":
		if m.detailSelected > 0 {
			m.detailSelected--
		}
	case "down", "j":
		if m.detailSelected < len(m.detailProjects)-1 {
			m.detailSelected++
		}
	case "b":
		if account != nil {
			cwd, err := os.Getwd()
			if err != nil {
				m.errText = err.Error()
				return m, nil
			}
			m.startBind(*account, cwd)
		}
	case "d":
		if account != nil && m.detailSelected < len(m.detailProjects) {
			project := m.detailProjects[m.detailSelected]
			m.confirmPrompt = "解除绑定？\n\n" + project.Path + "\n\n解除后该文件夹会恢复绑定前的 Git 配置。"
			m.confirmAction = func() tea.Cmd { return m.unbindCommand(project.Path) }
			m.screen = screenConfirm
		}
	case "t":
		if account != nil {
			m.message = "正在检查 " + account.Alias + " …"
			m.busy = true
			return m, m.verifyCommand(*account)
		}
	case "r":
		if account != nil {
			m.busy = true
			m.message = "正在检查并修复配置…"
			return m, m.reconcileCommand(*account)
		}
	case "x":
		if account != nil {
			m.confirmPrompt = "删除账号 " + account.Alias + "？\n\n它绑定的项目会先被解除绑定。"
			m.confirmAction = func() tea.Cmd { return m.removeAccountCommand(*account) }
			m.screen = screenConfirm
		}
	}
	return m, nil
}

func (m Model) handleLoginKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.login.step {
	case 0:
		switch msg.String() {
		case "esc":
			m.screen = screenAccounts
		case "up", "k":
			if m.login.providerIndex > 0 {
				m.login.providerIndex--
			}
		case "down", "j":
			if m.login.providerIndex < 2 {
				m.login.providerIndex++
			}
		case "enter":
			m.login.step = 1
			m.login.methodIndex = 0
		}
	case 1:
		switch msg.String() {
		case "esc":
			m.login.step = 0
		case "up", "k":
			if m.login.methodIndex > 0 {
				m.login.methodIndex--
			}
		case "down", "j":
			if m.login.methodIndex < 1 {
				m.login.methodIndex++
			}
		case "enter":
			if m.login.methodIndex == 0 {
				m.busy = true
				m.message = "正在使用本机已登录的账号…"
				return m, m.loginCommand(providerAt(m.login.providerIndex), "", true)
			}
			m.login.step = 2
			m.login.token = ""
		}
	case 2:
		switch msg.String() {
		case "esc":
			m.login.step = 1
			m.login.token = ""
		case "backspace":
			if len(m.login.token) > 0 {
				m.login.token = m.login.token[:len(m.login.token)-1]
			}
		case "enter":
			if strings.TrimSpace(m.login.token) == "" {
				m.errText = "请先粘贴访问码（在平台上创建 token 后复制）。"
				return m, nil
			}
			m.busy = true
			m.message = "正在登录…"
			return m, m.loginCommand(providerAt(m.login.providerIndex), m.login.token, false)
		default:
			if msg.Type == tea.KeyRunes {
				m.login.token += string(msg.Runes)
			}
		}
	}
	return m, nil
}

func providerAt(index int) domain.ProviderType {
	switch index {
	case 1:
		return domain.ProviderGitLab
	case 2:
		return domain.ProviderGitea
	default:
		return domain.ProviderGitHub
	}
}

func (m Model) handleBindKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.detailAccount != nil {
			m.screen = screenDetail
		} else {
			m.screen = screenAccounts
		}
		m.bind.manual = false
		return m, nil
	case "up", "k":
		if m.bind.selected > 0 {
			m.bind.selected--
		}
	case "down", "j":
		if m.bind.selected < len(m.bind.entries)+1 {
			m.bind.selected++
		}
	case "e":
		m.bind.manual = !m.bind.manual
	case "backspace":
		if m.bind.manual && len(m.bind.path) > 1 {
			m.bind.path = strings.TrimRight(m.bind.path[:len(m.bind.path)-1], "/")
			if m.bind.path == "" {
				m.bind.path = "/"
			}
		}
	case "enter":
		if m.bind.manual {
			return m.finishBind()
		}
		switch m.bind.selected {
		case 0: // bind this folder
			return m.finishBind()
		case 1: // go up
			parent := filepath.Dir(m.bind.path)
			entries, err := listDirs(parent)
			if err != nil {
				m.errText = "无法读取文件夹：" + err.Error()
				return m, nil
			}
			m.bind.path, m.bind.entries, m.bind.selected = parent, entries, 0
		default:
			name := m.bind.entries[m.bind.selected-2]
			next := filepath.Join(m.bind.path, name)
			entries, err := listDirs(next)
			if err != nil {
				m.errText = "无法读取文件夹：" + err.Error()
				return m, nil
			}
			m.bind.path, m.bind.entries, m.bind.selected = next, entries, 0
		}
	default:
		if m.bind.manual && msg.Type == tea.KeyRunes {
			m.bind.path += string(msg.Runes)
		}
	}
	return m, nil
}

func (m Model) finishBind() (tea.Model, tea.Cmd) {
	m.busy = true
	m.message = "正在绑定 " + m.bind.path + " …"
	return m, m.bindCommand(m.bind.account, m.bind.path)
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "enter":
		action := m.confirmAction
		m.confirmAction = nil
		m.screen = screenDetail
		if m.detailAccount == nil {
			m.screen = screenAccounts
		}
		if action != nil {
			return m, action()
		}
	case "n", "esc":
		m.confirmAction = nil
		if m.detailAccount != nil {
			m.screen = screenDetail
		} else {
			m.screen = screenAccounts
		}
	}
	return m, nil
}

// offerOnboardingBind guides a brand-new user: after the first successful
// login, offer to bind the folder they launched gitra from.
func (m *Model) offerOnboardingBind(alias string) tea.Cmd {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	status, err := m.app.Bindings.Status(m.ctx(), cwd)
	if err != nil || status.Bound {
		return nil
	}
	account, err := m.app.Accounts.GetByAlias(m.ctx(), alias)
	if err != nil {
		return nil
	}
	repositoryPath := status.Repository
	m.confirmPrompt = "检测到当前文件夹是一个 Git 仓库：\n\n" + repositoryPath +
		"\n\n要把它绑定到 " + alias + " 吗？"
	m.confirmAction = func() tea.Cmd { return m.bindCommand(account, repositoryPath) }
	m.screen = screenConfirm
	return nil
}

var _ = app.DeleteAccountRequest{}
