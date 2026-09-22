package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/zhanhd/gitra/internal/domain"
)

const (
	cardContentWidth = 34
	cardTotalWidth   = cardContentWidth + 4 // border + padding
	cardGap          = 2
)

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

// View implements tea.Model.
func (m Model) View() string {
	if m.quit {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("gitra"))
	builder.WriteString(subtitleStyle.Render("  让每个文件夹用对账号"))
	builder.WriteString("\n\n")

	switch m.screen {
	case screenAccounts:
		builder.WriteString(m.viewAccounts())
	case screenDetail:
		builder.WriteString(m.viewDetail())
	case screenLogin:
		builder.WriteString(m.viewLogin())
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
	builder.WriteString("\n\n" + helpStyle.Render(m.helpLine()))
	return builder.String()
}

func (m Model) viewAccounts() string {
	if len(m.accounts) == 0 {
		var builder strings.Builder
		builder.WriteString(accentStyle.Render("欢迎使用 gitra 👋") + "\n\n")
		builder.WriteString("你还没有登录任何 Git 账号。请选择（↑↓ 移动，回车确认，鼠标可直接点击）：\n\n")
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
	body += "\n" + subtitleStyle.Render("[+ 添加账号]") + "\n"
	return body
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
	for index, project := range m.detailProjects {
		marker := "  "
		if index == m.detailSelected {
			marker = "> "
		}
		state := healthLabel(project.State)
		line := fmt.Sprintf("%s%s\n    %s  %s\n", marker, project.Path, project.State, state)
		if index == m.detailSelected {
			builder.WriteString(selectedItem.Render(line))
		} else {
			builder.WriteString(line)
		}
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
		builder.WriteString(fmt.Sprintf("登录 %s（↑↓ 选择，回车继续，Esc 返回）\n\n", providerLabel(providerAt(m.login.providerIndex))))
		options := []string{
			"使用本机已登录的 gh / glab（推荐，无需输入）",
			"粘贴访问码（在平台网页上创建 token 后复制）",
		}
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
	case 2:
		builder.WriteString(fmt.Sprintf("粘贴访问码（%s）\n\n", providerLabel(providerAt(m.login.providerIndex))))
		masked := strings.Repeat("•", len(m.login.token))
		builder.WriteString("> " + masked + "▌\n\n")
		builder.WriteString(subtitleStyle.Render("在平台网页创建 token（GitHub: Settings → Developer settings → Tokens，\n"+
			"权限勾选 repo / read:user / user:email），复制后在此粘贴并按回车。\n") + "\n")
	}
	return builder.String()
}

func (m Model) viewBind() string {
	var builder strings.Builder
	builder.WriteString("选择要绑定的项目文件夹\n\n")
	builder.WriteString("当前：")
	if m.bind.manual {
		builder.WriteString(accentStyle.Render(m.bind.path + "▌"))
	} else {
		builder.WriteString(accentStyle.Render(m.bind.path))
	}
	builder.WriteString("\n\n")
	if m.bind.manual {
		builder.WriteString(subtitleStyle.Render("手动输入模式：输入路径后回车绑定，按 E 返回列表选择。") + "\n")
		return builder.String()
	}
	options := append([]string{"✓ 使用这个文件夹", ".. （上一层）"}, m.bind.entries...)
	for index, option := range options {
		marker := "  "
		if index == m.bind.selected {
			marker = "> "
		}
		if index == m.bind.selected {
			builder.WriteString(menuSelected.Render(marker+option) + "\n")
		} else {
			builder.WriteString(marker + option + "\n")
		}
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
			return "↑↓ 选择   Enter 确认   Q 退出   （也可以用鼠标直接点击上面的选项）"
		}
		return "↑↓←→ 选择   Enter 打开   A 添加账号   T 测试连接   Q 退出   （鼠标点击卡片即可打开）"
	case screenDetail:
		return "↑↓ 选择项目   B 绑定文件夹   D 解除绑定   T 测试连接   R 修复配置   X 删除账号   Esc 返回"
	case screenLogin:
		return "↑↓ 选择   Enter 继续   Esc 返回"
	case screenBind:
		return "↑↓ 选择   Enter 打开/绑定   E 手动输入路径   Esc 返回"
	case screenRemote:
		return "输入/粘贴仓库地址   Enter 确认并绑定   Esc 返回"
	case screenConfirm:
		return "Y 确认   N 取消   （也可用鼠标点击）"
	default:
		return ""
	}
}
