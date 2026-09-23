package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
)

// repaintFromScratch erases the screen and homes the cursor so the next frame
// is painted on a clean slate. Bubble Tea's own ClearScreen command proved a
// no-op in this configuration (verified byte-for-byte on a pty), so the
// sequence is written directly; the terminal receives one small write, and the
// renderer repaints on its next flush.
func repaintFromScratch() tea.Msg {
	_, _ = os.Stdout.WriteString("\x1b[2J\x1b[H")
	return nil
}

// Run starts the interactive TUI.
func Run(application *bootstrap.App) error {
	// Mouse events are deliberately NOT captured: the terminal keeps its
	// native drag-to-select/copy behaviour and a stray click cannot trigger
	// an action. The TUI is keyboard-driven (baseline §49).
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
		// Resizing re-wraps the frame that is already on screen, so the
		// renderer's line bookkeeping no longer matches the terminal and stale
		// lines stay behind (the duplicated help bar users hit). Erase the
		// screen and repaint from scratch.
		return m, repaintFromScratch

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quit = true
			return m, tea.Quit
		}
		if m.busy {
			// Long operations (login, connection test) must be escapable: the
			// work continues in the background, its result is still applied.
			if msg.String() == "esc" {
				m.busy = false
				m.message = ""
				m.errText = "已停止等待。如果刚才的操作稍后完成，结果仍会出现在这里。"
			}
			return m, nil
		}
		return m.handleKey(msg)

	case candidatesMsg:
		m.login.detecting = false
		m.login.candidates = msg.candidates
		if m.login.autoPickCLI {
			m.login.autoPickCLI = false
			for index, candidate := range m.login.candidates {
				if candidate.Kind == "cli" {
					m.login.methodIndex = index
					return m.activateLoginOption(index)
				}
			}
			m.errText = "浏览器授权已完成，但没有读到可用的登录会话，请重试。"
		}
		if m.login.methodIndex >= len(m.loginOptions()) {
			m.login.methodIndex = 0
		}
		return m, nil

	case browserLoginRequestedMsg:
		providerType := msg.provider
		if _, ok := browserLoginCLI(providerType); !ok {
			m.errText = "这台电脑没有官方 GitHub 客户端。可以改用「粘贴访问码」，或先安装 gh（可选）。"
			m.screen = screenLogin
			return m, nil
		}
		host := defaultHost(providerType)
		m.message = "请在浏览器里完成授权…"
		m.busy = true
		m.screen = screenLogin
		return m, m.browserLoginCommand(providerType, host)

	case keyLoadedMsg:
		m.busy = false
		if msg.err != nil {
			providerType := m.login.keyPick.provider
			m.confirmPrompt = fmt.Sprintf(
				"密钥没有加载成功（口令可能输错了）。\n\n要改用浏览器登录 %s 吗？（不需要密钥，点一次「授权」就好）",
				providerLabel(providerType))
			m.confirmAction = func() tea.Cmd {
				return func() tea.Msg { return browserLoginRequestedMsg{provider: providerType} }
			}
			m.screen = screenConfirm
			return m, nil
		}
		m.busy = true
		m.message = "正在用这把密钥验证身份…"
		return m, m.verifyKeyCommand(msg.path)

	case keyVerifiedMsg:
		m.busy = false
		if !msg.ok {
			// Never ask the user to copy a public key around: offer the
			// zero-knowledge alternative instead (browser authorization).
			providerType := m.login.keyPick.provider
			m.confirmPrompt = fmt.Sprintf(
				"这把密钥没有被 %s 接受。\n\n要改用浏览器登录吗？（推荐：不需要密钥，点一次「授权」就好）",
				providerLabel(providerType))
			m.confirmAction = func() tea.Cmd {
				return func() tea.Msg { return browserLoginRequestedMsg{provider: providerType} }
			}
			m.screen = screenConfirm
			return m, nil
		}
		m.message = fmt.Sprintf("已验证：这把密钥属于 %s", msg.candidate.Username)
		m.busy = true
		return m, m.sshKeyLoginCommand(msg.candidate)

	case cliLoginResultMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = "浏览器登录没有完成：" + plainError(msg.err)
			return m, nil
		}
		m.message = "已授权，正在读取账号…"
		m.login.detecting = true
		m.login.autoPickCLI = true
		return m, m.detectCommand(providerAt(m.login.providerIndex), "")

	case openBrowserMsg:
		if msg.err != nil {
			m.message = "如果浏览器没有自动打开，请手动访问：" + msg.url
		}
		return m, nil

	case loginDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = loginErrorText(msg)
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
		return m, m.offerFirstProject(msg.alias)

	case bindDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		if msg.path != "" {
			m.message = "已绑定：" + msg.path + " → " + msg.alias
			m.errText = ""
			m.lastBindDir = filepath.Dir(msg.path)
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
		if msg.createdRemote {
			path := msg.path
			m.confirmPrompt = "远端仓库已创建并绑定。\n\n要现在把本地代码首次上传吗？\n（只需这一次，之后在编辑器里同步即可）"
			m.confirmAction = func() tea.Cmd { return m.publishCommand(path) }
			m.screen = screenConfirm
		}
		return m, nil

	case bindManyDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		bound, already, skipped, failed := 0, 0, 0, 0
		var details []string
		for _, result := range msg.results {
			reason := result.Reason
			switch result.Status {
			case "bound":
				bound++
			case "already":
				already++
			case "skipped":
				skipped++
				details = append(details, filepath.Base(result.Path)+"："+reason)
			default:
				failed++
				details = append(details, filepath.Base(result.Path)+"："+reason)
			}
		}
		m.message = fmt.Sprintf("批量绑定完成：成功 %d 个", bound)
		if already > 0 {
			m.message += fmt.Sprintf("，已绑定 %d 个", already)
		}
		if skipped > 0 {
			m.message += fmt.Sprintf("，跳过 %d 个", skipped)
		}
		if failed > 0 {
			m.message += fmt.Sprintf("，失败 %d 个", failed)
		}
		if len(details) > 8 {
			details = append(details[:8], fmt.Sprintf("…还有 %d 条，见下方项目状态", len(details)-8))
		}
		m.errText = strings.Join(details, "\n")
		m.lastBindDir = m.bind.path
		m.bind.marked = map[string]bool{}
		if m.detailAccount != nil {
			m.loadDetailProjects(*m.detailAccount)
		}
		m.loadAccounts()
		if m.detailAccount != nil {
			m.screen = screenDetail
		} else {
			m.screen = screenAccounts
		}
		return m, nil

	case publishDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		m.message = "已上传：" + msg.branch + " → origin（已在远端确认）"
		m.errText = ""
		if m.detailAccount != nil {
			m.loadDetailProjects(*m.detailAccount)
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

	case remoteCheckMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = plainError(msg.err)
			return m, nil
		}
		if msg.check.Exists {
			m.message = "已确认仓库存在，正在绑定…"
			m.busy = true
			return m, m.bindOnlyCommand(m.bind.account, m.bind.path)
		}
		m.confirmPrompt = fmt.Sprintf(
			"这个仓库在 %s 上还不存在：\n\n%s/%s\n\n要现在创建吗？\n（默认创建为私有仓库，之后可在网页改成公开）",
			providerLabel(m.bind.account.Provider.Type), msg.check.Owner, msg.check.Name)
		account, path := m.bind.account, m.bind.path
		m.confirmAction = func() tea.Cmd { return m.createAndBindCommand(account, path) }
		m.screen = screenConfirm
		return m, nil

	case openBindPickerMsg:
		m.startBind(msg.account, msg.path)
		return m, nil

	case quitConfirmedMsg:
		m.quit = true
		return m, tea.Quit

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

// loginErrorText turns a failed login into an actionable, TUI-friendly hint
// (never a shell command).
func loginErrorText(msg loginDoneMsg) string {
	if errors.Is(msg.err, domain.ErrAuthInvalid) {
		if msg.usedToken {
			return "访问码无效或权限不足：请确认 token 权限包含 repo / read:user / user:email。"
		}
		return "本机没有可复用的登录：请选择「粘贴访问码」，或先在本机登录 gh 后重试。"
	}
	return plainError(msg.err)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenAccounts:
		return m.handleAccountsKey(msg)
	case screenDetail:
		return m.handleDetailKey(msg)
	case screenLogin:
		return m.handleLoginKey(msg)
	case screenKeyPick:
		return m.handleKeyPickKey(msg)
	case screenBind:
		return m.handleBindKey(msg)
	case screenRemote:
		return m.handleRemoteKey(msg)
	case screenConfirm:
		return m.handleConfirmKey(msg)
	}
	return m, nil
}

func (m Model) handleAccountsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.accounts) == 0 {
		items := welcomeItems()
		switch msg.String() {
		case "q":
			return m.requestQuit()
		case "up", "k":
			if m.welcomeIndex > 0 {
				m.welcomeIndex--
			}
		case "down", "j":
			if m.welcomeIndex < len(items)-1 {
				m.welcomeIndex++
			}
		case "enter":
			return m.activateWelcome(items[m.welcomeIndex])
		case "a":
			m.message, m.errText = "", ""
			m.login = loginState{step: 1, detecting: true}
			m.screen = screenLogin
			return m, m.detectCommand(domain.ProviderGitHub, "")
		}
		return m, nil
	}
	columns := m.columns()
	switch msg.String() {
	case "q":
		return m.requestQuit()
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
	case "b":
		if len(m.accounts) == 0 {
			m.errText = "还没有账号，先按 A 登录一个。"
			return m, nil
		}
		if account, ok := m.accountByID(m.accounts[m.selected].ID); ok {
			m.message, m.errText = "", ""
			m.startBind(account, suggestedBindDir())
		}
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
	case "u":
		if len(m.detailProjects) > 0 && m.detailSelected < len(m.detailProjects) {
			path := m.detailProjects[m.detailSelected].Path
			m.busy = true
			m.message = "正在上传（首次）…"
			return m, m.publishCommand(path)
		}
		m.errText = "先选择一个项目，再按 U 上传。"
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
		options := m.loginOptions()
		switch msg.String() {
		case "esc":
			m.login.step = 0
			m.login.candidates = nil
		case "up", "k":
			if m.login.methodIndex > 0 {
				m.login.methodIndex--
			}
		case "down", "j":
			if m.login.methodIndex < len(options)-1 {
				m.login.methodIndex++
			}
		case "enter":
			return m.activateLoginOption(m.login.methodIndex)
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

// activateWelcome starts login for the chosen provider; quitting always asks
// for confirmation first so a stray click cannot close the app.
func (m Model) activateWelcome(item welcomeItem) (tea.Model, tea.Cmd) {
	if item.quit {
		return m.requestQuit()
	}
	m.message, m.errText = "", ""
	index := 0
	for position, candidate := range welcomeItems() {
		if candidate.provider == item.provider && !candidate.quit {
			index = position
			break
		}
	}
	m.login = loginState{providerIndex: index, step: 1, methodIndex: 0, detecting: true}
	m.screen = screenLogin
	return m, m.detectCommand(providerAt(index), "")
}

// activateLoginOption runs the chosen discovered login, or falls back to the
// manual access code (which opens the prefilled token page).
func (m Model) activateLoginOption(index int) (tea.Model, tea.Cmd) {
	options := m.loginOptionList()
	if index < 0 || index >= len(options) {
		return m, nil
	}
	option := options[index]
	switch option.kind {
	case loginOptionCandidate:
		m.busy = true
		if option.candidate.Kind == "ssh-key" {
			m.message = "正在使用已有密钥 " + filepath.Base(option.candidate.KeyPath) + " …"
			return m, m.sshKeyLoginCommand(option.candidate)
		}
		m.message = "正在使用本机已登录的账号…"
		return m, m.loginCommand(providerAt(m.login.providerIndex), "", true)
	case loginOptionSSHKey:
		providerType := providerAt(m.login.providerIndex)
		host := defaultHost(providerType)
		keys := []app.SSHKeyInfo{}
		if m.app.Detector != nil {
			keys = m.app.Detector.SSHKeyInfos()
		}
		m.keys = keys
		m.keyIndex = 0
		m.login.keyPick = keyPickState{provider: providerType, host: host}
		m.message, m.errText = "", ""
		m.screen = screenKeyPick
		return m, nil
	case loginOptionBrowser:
		providerType := providerAt(m.login.providerIndex)
		host := defaultHost(providerType)
		m.message = "请在浏览器里完成授权…"
		m.busy = true
		return m, m.browserLoginCommand(providerType, host)
	default:
		m.login.step = 2
		m.login.token = ""
		url := tokenPageURL(providerAt(m.login.providerIndex), "")
		m.message = "已打开创建访问码的页面"
		return m, openBrowserCmd(url)
	}
}

// loginOptionList is the single source of truth for the login screen rows.
func (m Model) loginOptionList() []loginOption {
	options := make([]loginOption, 0, len(m.login.candidates)+2)
	for _, candidate := range m.login.candidates {
		options = append(options, loginOption{kind: loginOptionCandidate, label: candidate.Label, candidate: candidate})
	}
	providerType := providerAt(m.login.providerIndex)
	if _, ok := browserLoginCLI(providerType); ok {
		client := "gh"
		if providerType == domain.ProviderGitLab {
			client = "glab"
		}
		options = append(options, loginOption{
			kind:  loginOptionBrowser,
			label: fmt.Sprintf("在浏览器里登录 %s（使用官方 %s 客户端，推荐）", providerLabel(providerType), client),
		})
	}
	options = append(options,
		loginOption{kind: loginOptionSSHKey, label: "使用本地 SSH 密钥…（带口令的会在这里让你输入一次）"},
		loginOption{kind: loginOptionManual, label: "粘贴访问码（高级：自己去网页创建）"},
	)
	return options
}

func defaultHost(providerType domain.ProviderType) string {
	if endpoint, ok := domain.DefaultEndpoint(providerType); ok {
		return endpoint.Host
	}
	return ""
}

// requestQuit opens the confirmation dialog instead of quitting immediately.
func (m Model) requestQuit() (tea.Model, tea.Cmd) {
	m.confirmPrompt = "要退出 gitra 吗？\n\n已绑定的文件夹不受影响，随时可以再打开。"
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg { return quitConfirmedMsg{} }
	}
	m.screen = screenConfirm
	return m, nil
}

// handleKeyPickKey drives the SSH key picker.
func (m Model) handleKeyPickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenLogin
		m.errText = ""
		return m, nil
	case "up", "k":
		if m.keyIndex > 0 {
			m.keyIndex--
		}
	case "down", "j":
		if m.keyIndex < len(m.keys)-1 {
			m.keyIndex++
		}
	case "enter":
		if len(m.keys) == 0 {
			m.errText = "没有找到可用的 SSH 私钥（~/.ssh 下没有 id_* 之类的文件）。"
			return m, nil
		}
		key := m.keys[m.keyIndex]
		if key.NeedsPassphrase {
			m.busy = true
			m.message = "这把密钥需要口令：请在下方输入一次（macOS 会存进钥匙串）"
			return m, m.loadKeyCommand(key.Path)
		}
		m.busy = true
		m.message = "正在用这把密钥验证身份…"
		return m, m.verifyKeyCommand(key.Path)
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
	if m.bind.manual {
		return m.handleBindManualKey(msg)
	}
	options := m.bindOptionList()
	switch msg.String() {
	case "esc", "left", "h", "backspace":
		// Inside the picker, going up one level is what users expect from Esc.
		// Only at the directory the picker was opened in does it leave.
		if m.bind.startPath != "" && m.bind.path != m.bind.startPath && m.bind.path != "/" {
			return m.goUpOneLevel()
		}
		if msg.String() == "esc" {
			m.lastBindDir = m.bind.path
			if m.detailAccount != nil {
				m.screen = screenDetail
			} else {
				m.screen = screenAccounts
			}
			m.message, m.errText = "", ""
		}
		return m, nil
	case "up", "k":
		if m.bind.selected > 0 {
			m.bind.selected--
		}
	case "down", "j":
		if m.bind.selected < len(options)-1 {
			m.bind.selected++
		}
	case " ":
		m.toggleMark()
	case "b":
		return m.prepareBind()
	case "e":
		m.bind.manual = true
	case "enter":
		if m.bind.selected < 0 || m.bind.selected >= len(options) {
			return m, nil
		}
		option := options[m.bind.selected]
		switch option.kind {
		case bindOptionBulk:
			paths := m.markedPaths()
			if len(paths) == 0 {
				return m, nil
			}
			m.busy = true
			m.message = fmt.Sprintf("正在绑定 %d 个文件夹…", len(paths))
			m.errText = ""
			return m, m.bindManyCommand(m.bind.account, paths)
		case bindOptionCurrent:
			return m.prepareBind()
		case bindOptionUp:
			return m.goUpOneLevel()
		case bindOptionDir:
			next := filepath.Join(m.bind.path, option.name)
			entries, err := listDirs(next)
			if err != nil {
				m.errText = "无法读取文件夹：" + err.Error()
				return m, nil
			}
			m.bind.path, m.bind.entries, m.bind.selected = next, entries, 0
		}
	}
	return m, nil
}

// goUpOneLevel moves the picker to the parent directory.
func (m Model) goUpOneLevel() (tea.Model, tea.Cmd) {
	parent := filepath.Dir(m.bind.path)
	if parent == m.bind.path {
		return m, nil // already at the filesystem root
	}
	entries, err := listDirs(parent)
	if err != nil {
		m.errText = "无法读取文件夹：" + err.Error()
		return m, nil
	}
	m.bind.path, m.bind.entries, m.bind.selected = parent, entries, 0
	m.errText = ""
	return m, nil
}

func (m Model) handleBindManualKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.bind.manual = false
	case "backspace":
		if len(m.bind.path) > 1 {
			m.bind.path = strings.TrimRight(m.bind.path[:len(m.bind.path)-1], "/")
			if m.bind.path == "" {
				m.bind.path = "/"
			}
		}
	case "enter":
		m.bind.manual = false
		return m.prepareBind()
	default:
		if msg.Type == tea.KeyRunes {
			m.bind.path += string(msg.Runes)
		}
	}
	return m, nil
}

// bindOptionList is the single source of truth for the picker rows.
func (m Model) bindOptionList() []bindOption {
	options := make([]bindOption, 0, len(m.bind.entries)+3)
	if count := len(m.markedPaths()); count > 0 {
		options = append(options, bindOption{
			kind:  bindOptionBulk,
			label: fmt.Sprintf("✓ 绑定已选的 %d 个文件夹（回车执行）", count),
		})
	}
	options = append(options,
		bindOption{kind: bindOptionCurrent, label: "使用这个文件夹（回车＝绑定它）"},
		bindOption{kind: bindOptionUp, label: ".. （上一层，也可以按 Esc 返回）"},
	)
	for _, name := range m.bind.entries {
		label := name
		if m.bind.marked[filepath.Join(m.bind.path, name)] {
			label = "✓ " + name
		}
		options = append(options, bindOption{kind: bindOptionDir, label: label, name: name})
	}
	return options
}

// toggleMark marks or unmarks the folder under the cursor.
func (m *Model) toggleMark() {
	options := m.bindOptionList()
	if m.bind.selected < 0 || m.bind.selected >= len(options) {
		return
	}
	option := options[m.bind.selected]
	if option.kind != bindOptionDir {
		m.errText = "空格用于标记文件夹：把光标移到子文件夹上再按空格，然后选「绑定已选的 N 个文件夹」。"
		return
	}
	path := filepath.Join(m.bind.path, option.name)
	if m.bind.marked[path] {
		delete(m.bind.marked, path)
	} else {
		if m.bind.marked == nil {
			m.bind.marked = map[string]bool{}
		}
		m.bind.marked[path] = true
	}
	m.errText = ""
}

// markedPaths returns the marked folders in listing order.
func (m Model) markedPaths() []string {
	if len(m.bind.marked) == 0 {
		return nil
	}
	paths := make([]string, 0, len(m.bind.marked))
	for path := range m.bind.marked {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (m Model) finishBind() (tea.Model, tea.Cmd) {
	m.busy = true
	m.message = "正在绑定 " + m.bind.path + " …"
	return m, m.bindCommand(m.bind.account, m.bind.path)
}

// prepareBind checks whether the folder still needs a repository address before
// binding it.
func (m Model) prepareBind() (tea.Model, tea.Cmd) {
	status, err := m.app.Bindings.Status(m.ctx(), m.bind.path)
	if err != nil {
		m.errText = plainError(err)
		return m, nil
	}
	if status.HasOrigin {
		return m.finishBind()
	}
	guess := "https://" + m.bind.account.Provider.Endpoint.Host + "/" + m.bind.account.Provider.Username + "/" + filepath.Base(status.Repository) + ".git"
	m.bind.needRemote = true
	m.bind.remoteURL = guess
	m.screen = screenRemote
	return m, nil
}

func (m Model) handleRemoteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenBind
	case "backspace":
		if len(m.bind.remoteURL) > 0 {
			m.bind.remoteURL = m.bind.remoteURL[:len(m.bind.remoteURL)-1]
		}
	case "enter":
		if strings.TrimSpace(m.bind.remoteURL) == "" {
			m.errText = "请粘贴仓库地址（在平台网页上复制 https://… 地址）。"
			return m, nil
		}
		m.busy = true
		m.message = "正在校验仓库地址…"
		return m, m.ensureAndCheckRemoteCommand(m.bind.account, m.bind.path, strings.TrimSpace(m.bind.remoteURL))
	default:
		if msg.Type == tea.KeyRunes {
			m.bind.remoteURL += string(msg.Runes)
		}
	}
	return m, nil
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

// offerFirstProject guides the user right after login: bind the folder they
// launched gitra from when it is an unbound repository, otherwise ask whether
// to pick a project folder now.
func (m *Model) offerFirstProject(alias string) tea.Cmd {
	account, err := m.app.Accounts.GetByAlias(m.ctx(), alias)
	if err != nil {
		return nil
	}

	if cwd, err := os.Getwd(); err == nil {
		if status, err := m.app.Bindings.Status(m.ctx(), cwd); err == nil && !status.Bound {
			repositoryPath := status.Repository
			m.confirmPrompt = "检测到当前文件夹是一个 Git 仓库：\n\n" + repositoryPath +
				"\n\n要把它绑定到 " + alias + " 吗？"
			m.confirmAction = func() tea.Cmd { return m.bindCommand(account, repositoryPath) }
			m.screen = screenConfirm
			return nil
		}
	}

	m.confirmPrompt = "已登录 " + alias + "。\n\n要现在添加一个项目吗？\n" +
		"（也可以稍后按 B 选择文件夹绑定）"
	start := suggestedBindDir()
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			return openBindPickerMsg{account: account, path: start}
		}
	}
	m.screen = screenConfirm
	return nil
}

// openBindPickerMsg opens the folder picker for one account.
type openBindPickerMsg struct {
	account domain.Account
	path    string
}

// suggestedBindDir picks a friendly starting folder for the picker.
func suggestedBindDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/"
	}
	for _, candidate := range []string{"Projects", "projects", "code", "Developer", "Documents"} {
		path := filepath.Join(home, candidate)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return home
}

var _ = app.DeleteAccountRequest{}
