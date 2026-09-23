package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/version"
)

const (
	cardContentWidth = 34
	cardTotalWidth   = cardContentWidth + 4 // border + padding
	cardGap          = 2
)

// visibleRows is how many list rows fit on screen, leaving room for the
// header, the status lines and the help bar.
func (m Model) visibleRows() int {
	rows := m.height - 12
	if rows < 5 {
		rows = 5
	}
	if rows > 30 {
		rows = 30
	}
	return rows
}

// windowRange returns the slice of indices to render so the selected row is
// always visible (a picker must never hide entries off-screen).
func windowRange(total, selected, size int) (start, end int) {
	if total <= size {
		return 0, total
	}
	if selected < size/2 {
		return 0, size
	}
	if selected > total-size/2-1 {
		return total - size, total
	}
	return selected - size/2, selected - size/2 + size
}

// columnsFor implements the responsive card layout (baseline §42).
func columnsFor(terminalWidth int) int {
	usable := terminalWidth - 4
	if usable < cardTotalWidth {
		return 1
	}
	columns := usable / (cardTotalWidth + cardGap)
	if columns < 1 {
		columns = 1
	}
	if columns > 4 {
		columns = 4
	}
	return columns
}

func (m Model) columns() int { return columnsFor(m.width) }

// helpSegments returns the key hints for the current screen as separate items,
// so they can be wrapped across lines on narrow terminals.
func (m Model) helpSegments() []string {
	switch m.screen {
	case screenAccounts:
		if len(m.accounts) == 0 {
			return []string{"↑↓ 选择", "Enter 确认", "Q 退出"}
		}
		return []string{"↑↓←→ 选择", "Enter 打开", "A 添加账号", "B 绑定项目", "T 测试连接", "Q 退出"}
	case screenDetail:
		return []string{"↑↓ 选择项目", "B 绑定文件夹", "U 首次上传", "D 解除绑定", "T 测试连接", "R 修复配置", "X 删除账号", "Esc 返回"}
	case screenLogin:
		return []string{"↑↓ 选择", "Enter 确认", "Esc 返回"}
	case screenKeyPick:
		return []string{"↑↓ 选择密钥", "Enter 使用", "Esc 返回"}
	case screenBind:
		return []string{"↑↓ 移动", "Enter 进入/绑定", "空格 标记多个", "B 绑定当前文件夹", "E 粘贴路径", "Esc 返回"}
	case screenRemote:
		return []string{"输入/粘贴仓库地址", "Enter 确认并绑定", "Esc 返回"}
	case screenConfirm:
		return []string{"Y 确认", "N 取消"}
	default:
		return nil
	}
}

// renderHelp wraps the hints so no key hint is ever cut off by the window width.
func (m Model) renderHelp() string {
	width := m.width
	if width <= 0 {
		width = 80
	}
	var lines []string
	current := ""
	for _, segment := range m.helpSegments() {
		segment = truncateToWidth(segment, width-2)
		candidate := segment
		if current != "" {
			candidate = current + "   " + segment
		}
		if current != "" && displayWidth(candidate) > width-2 {
			lines = append(lines, current)
			current = segment
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	var builder strings.Builder
	for index, line := range lines {
		if index > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(line)
	}
	return builder.String()
}

func displayWidth(text string) int { return lipgloss.Width(text) }

// truncateToWidth cuts text to at most width cells, adding an ellipsis.
func truncateToWidth(text string, width int) string {
	if width <= 0 || displayWidth(text) <= width {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "…"
		if displayWidth(candidate) <= width {
			return candidate
		}
	}
	return ""
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quit {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("gitra"))
	builder.WriteString(subtitleStyle.Render("  让每个文件夹用对账号"))
	if version.Version != "dev" {
		builder.WriteString(subtitleStyle.Render("  " + version.Version))
	}
	builder.WriteString("\n\n")

	switch m.screen {
	case screenAccounts:
		builder.WriteString(m.viewAccounts())
	case screenDetail:
		builder.WriteString(m.viewDetail())
	case screenLogin:
		builder.WriteString(m.viewLogin())
	case screenKeyPick:
		builder.WriteString(m.viewKeyPick())
	case screenBind:
		builder.WriteString(m.viewBind())
	case screenRemote:
		builder.WriteString(m.viewRemote())
	case screenConfirm:
		builder.WriteString(m.viewConfirm())
	}

	if m.busy {
		builder.WriteString("\n" + accentStyle.Render("正在处理，请稍候…"))
	}
	if m.message != "" {
		builder.WriteString("\n" + okStyle.Render(m.message))
	}
	if m.errText != "" {
		builder.WriteString("\n" + errStyle.Render(m.errText))
	}
	builder.WriteString("\n\n" + helpStyle.Render(m.renderHelp()))
	return fitToWidth(builder.String(), m.width)
}

// fitToWidth wraps any line that would exceed the terminal width. Without this,
// a too-wide line makes the terminal wrap it physically while the TUI counts it
// as one line, which leaves duplicated leftovers on screen.
func fitToWidth(view string, width int) string {
	if width <= 0 {
		return view
	}
	wrapper := lipgloss.NewStyle().Width(width)
	lines := strings.Split(view, "\n")
	for index, line := range lines {
		if lipgloss.Width(line) > width {
			lines[index] = strings.TrimRight(wrapper.Render(line), " \n")
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewAccounts() string {
	if len(m.accounts) == 0 {
		var builder strings.Builder
		builder.WriteString(accentStyle.Render("欢迎使用 gitra 👋") + "\n\n")
		builder.WriteString("你还没有登录任何 Git 账号。请选择（↑↓ 移动，回车确认）：\n\n")
		for index, item := range welcomeItems() {
			marker := "   "
			line := marker + item.label + "\n"
			if index == m.welcomeIndex {
				builder.WriteString(menuSelected.Render(" > "+item.label) + "\n")
			} else {
				builder.WriteString(line)
			}
		}
		builder.WriteString("\n" + subtitleStyle.Render(
			"登录只需要一次：本机登录过 gh 时直接回车即可；否则粘贴一次访问码，\n"+
				"之后在编辑器里同步/上传即可，不需要任何命令。") + "\n")
		return builder.String()
	}
	columns := m.columns()
	var rows []string
	for start := 0; start < len(m.accounts); start += columns {
		end := start + columns
		if end > len(m.accounts) {
			end = len(m.accounts)
		}
		var cards []string
		for index := start; index < end; index++ {
			cards = append(cards, m.renderCard(m.accounts[index], index == m.selected))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards...))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return body + "\n"
}

// welcomeItem is one entry of the first-run menu.
type welcomeItem struct {
	label    string
	provider domain.ProviderType
	quit     bool
}

// welcomeItems is shared by rendering, keyboard and mouse handling.
func welcomeItems() []welcomeItem {
	return []welcomeItem{
		{label: "登录 GitHub", provider: domain.ProviderGitHub},
		{label: "登录 GitLab", provider: domain.ProviderGitLab},
		{label: "登录 Gitea", provider: domain.ProviderGitea},
		{label: "退出", quit: true},
	}
}

func (m Model) renderCard(card AccountCard, focused bool) string {
	stateText, stateKind := stateLabel(card.LocalState)
	state := okStyle.Render(stateText)
	switch stateKind {
	case "warn":
		state = warnStyle.Render(stateText)
	case "err":
		state = errStyle.Render(stateText)
	}
	body := strings.Join([]string{
		accentStyle.Render(card.Provider) + subtitleStyle.Render("  "+card.Host),
		"",
		lipgloss.NewStyle().Bold(true).Render(card.Alias),
		state,
		"",
		card.AuthLabel,
		fmt.Sprintf("%d 个项目", card.ProjectCount),
	}, "\n")

	style := cardStyle
	if focused {
		style = focusedCardStyle
	}
	return style.Width(cardContentWidth).Render(body) + strings.Repeat(" ", cardGap)
}

func (m Model) viewDetail() string {
	if m.detailAccount == nil {
		return "账号数据已失效，按 Esc 返回。"
	}
	account := m.detailAccount
	var builder strings.Builder
	builder.WriteString(accentStyle.Render(m.detailCard.Provider + " · " + account.Alias))
	builder.WriteString("\n\n")
	rows := [][2]string{
		{"平台", m.detailCard.Provider},
		{"账号", account.Provider.Username},
		{"站点", account.Provider.Endpoint.Host},
		{"提交身份", account.Identity.Name + " <" + account.Identity.Email + ">"},
		{"登录方式", m.detailCard.AuthLabel},
		{"状态", m.detailCard.LocalState},
	}
	for _, row := range rows {
		builder.WriteString(fmt.Sprintf("%-10s %s\n", row[0], row[1]))
	}
	builder.WriteString("\n" + accentStyle.Render("已绑定的项目") + "\n")
	if len(m.detailProjects) == 0 {
		builder.WriteString("  （还没有）按 B 选择文件夹绑定\n")
	}
	size := m.visibleRows() / 2
	start, end := windowRange(len(m.detailProjects), m.detailSelected, size)
	if start > 0 {
		builder.WriteString(subtitleStyle.Render(fmt.Sprintf("↑ 上面还有 %d 个项目", start)) + "\n")
	}
	for index := start; index < end; index++ {
		project := m.detailProjects[index]
		marker := "  "
		if index == m.detailSelected {
			marker = "> "
		}
		line := fmt.Sprintf("%s%s\n    %s（%s）\n", marker, project.Path, healthLabel(project.State), project.State)
		if index == m.detailSelected {
			builder.WriteString(selectedItem.Render(line))
		} else {
			builder.WriteString(line)
		}
	}
	if end < len(m.detailProjects) {
		builder.WriteString(subtitleStyle.Render(fmt.Sprintf("↓ 下面还有 %d 个项目", len(m.detailProjects)-end)) + "\n")
	}
	return builder.String()
}

func (m Model) viewLogin() string {
	var builder strings.Builder
	switch m.login.step {
	case 0:
		builder.WriteString("选择平台（↑↓ 选择，回车继续，Esc 返回）\n\n")
		providers := []string{"GitHub", "GitLab", "Gitea"}
		for index, name := range providers {
			marker := "  "
			if index == m.login.providerIndex {
				marker = "> "
			}
			if index == m.login.providerIndex {
				builder.WriteString(menuSelected.Render(marker+name) + "\n")
			} else {
				builder.WriteString(marker + name + "\n")
			}
		}
	case 1:
		builder.WriteString(fmt.Sprintf("登录 %s（↑↓ 选择，回车确认，Esc 返回）\n\n",
			providerLabel(providerAt(m.login.providerIndex))))
		if m.login.detecting {
			builder.WriteString(subtitleStyle.Render("正在检查这台电脑上已有的登录方式…") + "\n")
			break
		}
		if len(m.login.candidates) == 0 {
			builder.WriteString(subtitleStyle.Render("这台电脑上没有发现可直接使用的登录。") + "\n\n")
		}
		options := m.loginOptions()
		for index, option := range options {
			marker := "  "
			if index == m.login.methodIndex {
				marker = "> "
			}
			if index == m.login.methodIndex {
				builder.WriteString(menuSelected.Render(marker+option) + "\n")
			} else {
				builder.WriteString(marker + option + "\n")
			}
		}
		builder.WriteString("\n" + subtitleStyle.Render(m.loginHint()) + "\n")
	case 2:
		builder.WriteString(fmt.Sprintf("粘贴访问码（%s）\n\n", providerLabel(providerAt(m.login.providerIndex))))
		masked := strings.Repeat("•", len(m.login.token))
		builder.WriteString("> " + masked + "▌\n\n")
		builder.WriteString(subtitleStyle.Render("已为你打开创建页面：\n"+tokenPageURL(providerAt(m.login.providerIndex), "")+"\n"+
			"权限："+tokenScopes(providerAt(m.login.providerIndex))+"\n") + "\n")
		builder.WriteString("创建后复制整串访问码，回到这里按 ⌘V 粘贴，再按回车。\n")
	}
	return builder.String()
}

// loginOptions renders the rows produced by loginOptionList.
func (m Model) loginOptions() []string {
	list := m.loginOptionList()
	options := make([]string, 0, len(list))
	for _, option := range list {
		options = append(options, option.label)
	}
	return options
}

// tokenPageURL returns the provider's token-creation page, with scopes
// preselected where the provider supports it.
func tokenPageURL(providerType domain.ProviderType, host string) string {
	switch providerType {
	case domain.ProviderGitHub:
		return "https://github.com/settings/tokens/new?scopes=repo,read:user,user:email&description=gitra"
	case domain.ProviderGitLab:
		return "https://gitlab.com/-/user_settings/personal_access_tokens?name=gitra&scopes=api,read_user"
	default:
		if host == "" {
			host = "gitea.com"
		}
		return "https://" + host + "/user/settings/applications"
	}
}

func tokenScopes(providerType domain.ProviderType) string {
	switch providerType {
	case domain.ProviderGitHub:
		return "repo、read:user、user:email"
	case domain.ProviderGitLab:
		return "api、read_user"
	default:
		return "仓库读写权限"
	}
}

// loginHint explains what the highlighted option will do, including what the
// resulting credential can be used for.
func (m Model) loginHint() string {
	if _, ok := browserLoginCLI(providerAt(m.login.providerIndex)); ok {
		return "推荐第一个选项：浏览器里点一次「Authorize」即可，不需要创建或粘贴任何内容。\n" +
			"授权得到的凭据存在系统钥匙串，可用于 pull / push / 合并（本地 commit 本来就不需要凭据），随时可在平台设置里撤销。"
	}
	return "这台电脑上没有可直接复用的登录，也没有找到官方客户端。\n" +
		"「粘贴访问码」= 在平台上创建一个令牌（我们会打开页面并勾好权限）；\n" +
		"它存在系统钥匙串里，之后 pull / push / 合并都由 git 自动使用，不需要再输入。"
}

func (m Model) viewKeyPick() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("使用本地 SSH 密钥登录 %s\n\n",
		providerLabel(m.login.keyPick.provider)))
	if len(m.keys) == 0 {
		builder.WriteString(subtitleStyle.Render("没有在 ~/.ssh 下找到可用的私钥。") + "\n")
		return builder.String()
	}
	if len(m.keys) > 0 && !m.keys[0].NeedsPassphrase {
		builder.WriteString(subtitleStyle.Render("已默认选中可直接使用的密钥；带口令的密钥排在最下面，会先让你输入一次。") + "\n\n")
	} else {
		builder.WriteString(subtitleStyle.Render("需要口令的密钥会先让你输入一次（macOS 会存进钥匙串，之后不再问）。") + "\n\n")
	}
	for index, key := range m.keys {
		marker := "  "
		if index == m.keyIndex {
			marker = "> "
		}
		label := key.Name
		if key.NeedsPassphrase {
			label += "（需要口令）"
		}
		if index == m.keyIndex {
			builder.WriteString(menuSelected.Render(marker+label) + "\n")
		} else {
			builder.WriteString(marker + label + "\n")
		}
	}
	return builder.String()
}

func (m Model) viewBind() string {
	var builder strings.Builder
	builder.WriteString("选择要绑定的项目文件夹\n\n")
	builder.WriteString(subtitleStyle.Render(
		"回车 = 进入文件夹；在「使用这个文件夹」上回车 = 绑定它。\n"+
			"空格 = 标记多个文件夹，标记后选「绑定已选的 N 个文件夹」可一次绑定。") + "\n\n")
	builder.WriteString("当前：" + accentStyle.Render(m.bind.path) + "\n\n")
	if m.bind.manual {
		builder.WriteString(accentStyle.Render(m.bind.path+"▌") + "\n")
		builder.WriteString(subtitleStyle.Render("手动输入模式：输入路径后回车绑定，按 Esc 返回列表选择。") + "\n")
		return builder.String()
	}

	options := m.bindOptionList()
	size := m.visibleRows()
	start, end := windowRange(len(options), m.bind.selected, size)
	if start > 0 {
		builder.WriteString(subtitleStyle.Render(fmt.Sprintf("↑ 上面还有 %d 项", start)) + "\n")
	}
	for index := start; index < end; index++ {
		marker := "  "
		if index == m.bind.selected {
			marker = "> "
		}
		if index == m.bind.selected {
			builder.WriteString(menuSelected.Render(marker+options[index].label) + "\n")
		} else {
			builder.WriteString(marker + options[index].label + "\n")
		}
	}
	if end < len(options) {
		builder.WriteString(subtitleStyle.Render(fmt.Sprintf("↓ 下面还有 %d 项", len(options)-end)) + "\n")
	}
	return builder.String()
}

func (m Model) viewRemote() string {
	var builder strings.Builder
	builder.WriteString("这个文件夹还没有仓库地址\n\n")
	builder.WriteString(subtitleStyle.Render("在平台网页上创建仓库并复制它的地址（https://…），粘贴到下面。\n"+
		"已有的地址不会被修改；这里只补一个还没有的 origin。\n") + "\n")
	builder.WriteString("> " + accentStyle.Render(m.bind.remoteURL+"▌") + "\n\n")
	builder.WriteString(subtitleStyle.Render("回车确认并绑定，Esc 返回。") + "\n")
	return builder.String()
}

func (m Model) viewConfirm() string {
	return dialogStyle.Render(m.confirmPrompt+"\n\n[Y] 确认\n[N] 取消") + "\n"
}

func (m Model) helpLine() string {
	switch m.screen {
	case screenAccounts:
		if len(m.accounts) == 0 {
			return "↑↓ 选择   Enter 确认   Q 退出"
		}
		return "↑↓←→ 选择   Enter 打开   A 添加账号   B 绑定项目   T 测试连接   Q 退出"
	case screenDetail:
		return "↑↓ 选择项目   B 绑定文件夹   D 解除绑定   T 测试连接   R 修复配置   X 删除账号   Esc 返回"
	case screenLogin:
		return "↑↓ 选择   Enter 确认   Esc 返回"
	case screenBind:
		return "↑↓ 移动   Enter 进入文件夹（在「使用这个文件夹」上＝绑定）   E 粘贴路径   Esc 返回"
	case screenRemote:
		return "输入/粘贴仓库地址   Enter 确认并绑定   Esc 返回"
	case screenConfirm:
		return "Y 确认   N 取消"
	default:
		return ""
	}
}
