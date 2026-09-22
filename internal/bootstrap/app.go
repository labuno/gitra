// Package bootstrap is the single composition root (baseline §24): it wires
// concrete adapters and strategies into the application services.
package bootstrap

import (
	"path/filepath"
	"time"

	"github.com/zhanhd/gitra/internal/adapters/gitcli"
	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/adapters/storage"
	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
	"github.com/zhanhd/gitra/internal/strategies/routing"
	"github.com/zhanhd/gitra/internal/strategies/routing/repolocal"
)

// App is the wired application: ports plus the services built on them.
type App struct {
	Deps       app.Deps
	Accounts   *app.AccountService
	Bindings   *app.BindingService
	Reconciler *app.Reconciler
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

	authRegistry := auth.NewRegistry()
	if err := authRegistry.Register(sshkey.New()); err != nil {
		return nil, err
	}
	routingRegistry := routing.NewRegistry()
	if err := routingRegistry.Register(repolocal.New()); err != nil {
		return nil, err
	}

	deps := app.Deps{
		Accounts:  storage.NewAccountStore(configDir),
		Bindings:  storage.NewBindingStore(configDir),
		Git:       gitcli.New(runner.New()),
		Snapshots: storage.NewSnapshotStore(),
		Auth:      authRegistry,
		Routing:   routingRegistry,
		Locker:    storage.NewFileLocker(filepath.Join(configDir, ".lock")),
		Clock:     systemClock{},
	}

	bindings := app.NewBindingService(deps)
	return &App{
		Deps:       deps,
		Accounts:   app.NewAccountService(deps, bindings),
		Bindings:   bindings,
		Reconciler: app.NewReconciler(deps),
	}, nil
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }
