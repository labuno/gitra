package tui

import "github.com/charmbracelet/lipgloss"

// Styles are deliberately simple: a user-facing tool, not a dashboard.
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)

	focusedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(0, 1)
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	accentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	dialogStyle  = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).Padding(1, 2)
	selectedItem = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
)
