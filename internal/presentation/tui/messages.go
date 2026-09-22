package tui

import "github.com/zhanhd/gitra/internal/app"

// Messages exchanged between tea.Cmd work and the Update loop (baseline §40).
type accountsLoadedMsg struct {
	cards []AccountCard
	err   error
}

type detailLoadedMsg struct {
	account  *AccountCard
	projects []Project
	err      error
}

type loginDoneMsg struct {
	alias     string
	username  string
	host      string
	created   bool
	usedToken bool
	err       error
}

type bindDoneMsg struct {
	path  string
	alias string
	// createdRemote is set when gitra created the provider-side repository in
	// this step, which makes an immediate first upload the natural next action.
	createdRemote bool
	err           error
}

// publishDoneMsg reports the outcome of the one-time first upload.
type publishDoneMsg struct {
	path   string
	branch string
	err    error
}

type unbindDoneMsg struct {
	path string
	err  error
}

type verifyDoneMsg struct {
	success bool
	status  string
	message string
}

type accountRemovedMsg struct {
	alias string
	err   error
}

// candidatesMsg carries the discovered logins for the current provider.
type candidatesMsg struct {
	candidates []app.Candidate
	err        error
}

// openBrowserMsg reports the result of launching the token page.
type openBrowserMsg struct {
	url string
	err error
}

// quitConfirmedMsg fires after the user confirms quitting.
type quitConfirmedMsg struct{}

// cliLoginResultMsg reports the outcome of handing the terminal to the
// provider's official CLI for a browser login.
type cliLoginResultMsg struct {
	err error
}

// remoteCheckMsg reports whether the provider already has the repository.
type remoteCheckMsg struct {
	check app.RemoteCheck
	err   error
}
