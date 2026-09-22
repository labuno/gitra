package tui

import (
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// openURL launches the system browser. Tests replace it.
var openURL = func(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		return openBrowserMsg{url: url, err: openURL(url)}
	}
}
