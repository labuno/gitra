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
	screenKeyPick
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

	// SSH key picker (loading a passphrase-protected key into ssh-agent).
	keys     []app.SSHKeyInfo
	keyIndex int

	// lastBindDir remembers where the user browsed, so the next picker opens
	// there instead of making them navigate again (defaults, not questions).
	lastBindDir string
	// confirm dialog
	confirmPrompt  string
	confirmAction  func() tea.Cmd
	confirmConfirm func() tea.Cmd

	quit bool
}

// keyPickState is the identity the chosen SSH key will be used for.
type keyPickState struct {
	provider domain.ProviderType
	host     string
}

type loginState struct {
	providerIndex int
	step          int // 0 provider, 1 discovered logins, 2 manual access code
	methodIndex   int
	token         string

	// Discovered reusable logins (gh session, existing SSH keys).
	detecting bool
	// autoPickCLI is set after a successful browser login: the freshly
	// authorized CLI session is then used without further questions.
	autoPickCLI bool
	candidates  []app.Candidate
	keyPick     keyPickState
}

// bindOptionKind distinguishes what a picker row does.
type bindOptionKind string

const (
	bindOptionBulk    bindOptionKind = "bulk"
	bindOptionCurrent bindOptionKind = "current"
	bindOptionUp      bindOptionKind = "up"
	bindOptionDir     bindOptionKind = "dir"
)

// bindOption is one row of the folder picker.
type bindOption struct {
	kind  bindOptionKind
	label string
	name  string
}

// loginOptionKind distinguishes what a login row does.
type loginOptionKind string

const (
	loginOptionCandidate loginOptionKind = "candidate"
	loginOptionBrowser   loginOptionKind = "browser"
	loginOptionSSHKey    loginOptionKind = "ssh-key"
	loginOptionManual    loginOptionKind = "manual"
)

// loginOption is one row of the login screen.
type loginOption struct {
	kind      loginOptionKind
	label     string
	candidate app.Candidate
}

type bindState struct {
	path     string
	entries  []string
	selected int
	account  domain.Account
	manual   bool // user is typing a path
	// startPath is where the picker was opened: Esc leaves the picker only
	// once we are back there (below it, Esc goes up one level).
	startPath string
	// marked holds folders selected for a bulk bind, keyed by absolute path so
	// marks survive navigating through subfolders.
	marked map[string]bool

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
