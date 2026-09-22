// Package bootstrap is the single composition root (baseline §24): it wires
// concrete adapters and strategies into the application services.
package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/zhanhd/gitra/internal/adapters/gitcli"
	"github.com/zhanhd/gitra/internal/adapters/provider/router"
	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/adapters/secretstore"
	"github.com/zhanhd/gitra/internal/adapters/sshcli"
	"github.com/zhanhd/gitra/internal/adapters/storage"
	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/httptoken"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"

	"github.com/zhanhd/gitra/internal/ports"
)

// App is the wired application: ports plus the services built on them.
type App struct {
	Deps         app.Deps
	Accounts     *app.AccountService
	Bindings     *app.BindingService
	Reconciler   *app.Reconciler
	Login        *app.LoginService
	Verification *app.VerificationService
}

// NewFromDeps builds every service around explicit dependencies. It is used by
// New and by tests that inject fake token/profile providers.
func NewFromDeps(deps app.Deps, tokens ports.TokenProvider, profiles ports.ProfileProvider, parsers ...ports.SSHIdentityParser) *App {
	bindings := app.NewBindingService(deps)
	accounts := app.NewAccountService(deps, bindings)
	var parser ports.SSHIdentityParser
	if len(parsers) > 0 {
		parser = parsers[0]
	}
	return &App{
		Deps:         deps,
		Accounts:     accounts,
		Bindings:     bindings,
		Reconciler:   app.NewReconciler(deps),
		Login:        app.NewLoginService(deps, accounts, tokens, profiles),
		Verification: app.NewVerificationService(deps, profiles, parser),
	}
}

// New assembles the application using the resolved config directory.
func New() (*App, error) {
	configDir, err := storage.ConfigDir()
	if err != nil {
		return nil, err
	}
	if err := storage.EnsureDir(configDir); err != nil {
		return nil, err
	}

	gitRunner := runner.New()
	resolvedHelper, err := secretstore.ResolveHelper(context.Background(), gitRunner, configDir)
	if err != nil {
		return nil, err
	}
	secrets := secretstore.NewGitCredentialStore(gitRunner, resolvedHelper)

	authRegistry := auth.NewRegistry()
	if err := authRegistry.Register(sshkey.New()); err != nil {
		return nil, err
	}
	if err := authRegistry.Register(httptoken.New(secrets)); err != nil {
		return nil, err
	}
	routingRegistry := routing.NewRegistry()
	if err := routingRegistry.Register(repolocal.New()); err != nil {
		return nil, err
	}

	deps := app.Deps{
		Accounts:  storage.NewAccountStore(configDir),
		Bindings:  storage.NewBindingStore(configDir),
		Git:       gitcli.New(gitRunner),
		Snapshots: storage.NewSnapshotStore(),
		Auth:      authRegistry,
		Routing:   routingRegistry,
		Locker:    storage.NewFileLocker(filepath.Join(configDir, ".lock")),
		Clock:     systemClock{},

		Secrets:           secrets,
		CredentialHelpers: secretstore.NewHelperResolver(gitRunner, configDir),
		SSH:               sshcli.New(gitRunner),
	}

	loginAdapter := router.New(gitRunner, os.Stdin)
	return NewFromDeps(deps, loginAdapter, loginAdapter, loginAdapter), nil
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }
