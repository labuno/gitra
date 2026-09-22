package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/bootstrap"
	"github.com/zhanhd/gitra/internal/domain"
)

type screen int

const (
	screenAccounts screen = iota
	screenDetail
	screenLogin
	screenBind
	screenRemote
	screenConfirm
)

// AccountCard is the home-screen card view model (baseline §39).
type AccountCard struct {
	ID           string
	Provider     string
	Alias        string
	Host         string
	AuthLabel    string
	LocalState   string // configured | needs_login | invalid
	ProjectCount int
}

// Project is one bound repository in the detail screen.
type Project struct {
	ID    string
	Alias string
	Path  string
	State string
}

// Model is the whole TUI state (baseline §37).
type Model struct {
	app *bootstrap.App

	width  int
	height int

	screen  screen
	busy    bool
	message string
	errText string

	accounts []AccountCard
	selected int
	// welcomeIndex drives the first-run menu shown when no account exists yet.
	welcomeIndex int

	detailAccount  *domain.Account
	detailCard     AccountCard
	detailProjects []Project
	detailSelected int

	login loginState
	bind  bindState
	// confirm dialog
	confirmPrompt string
	confirmAction func() tea.Cmd

	quit bool
}

type loginState struct {
	providerIndex int
	step          int // 0 provider, 1 discovered logins, 2 manual access code
	methodIndex   int
	token         string

	// Discovered reusable logins (gh session, existing SSH keys).
	detecting  bool
	candidates []app.Candidate
}

type bindState struct {
	path     string
	entries  []string
	selected int
	account  domain.Account
	manual   bool // user is typing a path

	// Remote step: some repositories have no origin yet. gitra asks for the
	// address once and never rewrites an existing remote.
	needRemote bool
	remoteURL  string
}

// services bundles the application services the TUI is allowed to call
// (baseline §38: no direct git/ssh/storage access from the presentation).
type services struct {
	app *bootstrap.App
}

func (s services) ctx() context.Context { return context.Background() }

// New builds the TUI model with local data loaded (no network on startup).
func New(app *bootstrap.App) Model {
	model := Model{app: app, screen: screenAccounts}
	model.loadAccounts()
	return model
}
