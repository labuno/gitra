// Package app contains application services. This file owns the binding
// lifecycle: snapshot, apply, verify, rollback, unbind and status.
package app

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/routing"
)

// Deps are the ports and strategies used by the binding service.
type Deps struct {
	Accounts  ports.AccountStore
	Bindings  ports.BindingStore
	Git       ports.Git
	Snapshots ports.SnapshotStore
	Auth      *auth.Registry
	Routing   *routing.Registry
	Locker    ports.Locker
	Clock     ports.Clock
}

// BindingService implements bind / unbind / status.
type BindingService struct {
	deps         Deps
	newBindingID func() (domain.BindingID, error)
}

// NewBindingService wires the service. Every write runs under the file lock.
func NewBindingService(deps Deps) *BindingService {
	return &BindingService{deps: deps, newBindingID: generateBindingID}
}

// BindRequest binds one repository path to one account.
type BindRequest struct {
	AccountID domain.AccountID
	Path      string
}

// UnbindRequest unbinds one repository path.
type UnbindRequest struct {
	Path string
}

// BindingStatus is the read model behind `gitra status`.
type BindingStatus struct {
	Bound      bool
	Repository string
	Binding    domain.RepositoryBinding
	Account    domain.Account
	Health     domain.BindingHealth
	Drift      []string
}

// managedKeys is the V1.0 ownership whitelist (baseline §9).
func managedKeys() []string {
	return []string{
		"user.name",
		"user.email",
		"core.sshCommand",
		"gitra.bindingId",
		"gitra.accountId",
		"gitra.strategy",
		"gitra.version",
	}
}

// Bind runs the fixed binding flow inside the write lock (baseline §6).
func (s *BindingService) Bind(ctx context.Context, req BindRequest) (domain.RepositoryBinding, error) {
	var binding domain.RepositoryBinding
	err := s.deps.Locker.WithWriteLock(ctx, func() error {
		created, err := s.bindLocked(ctx, req)
		if err != nil {
			return err
		}
		binding = created
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnsupportedRemote) {
			return domain.RepositoryBinding{}, fmt.Errorf(
				"%w\nRepository remote uses a non-SSH transport.\ngitra V1.0 core supports SSH remotes only; change the remote to git@<host>:<owner>/<repo>.git before binding.",
				err)
		}
		return domain.RepositoryBinding{}, err
	}
	return binding, nil
}

func (s *BindingService) bindLocked(ctx context.Context, req BindRequest) (domain.RepositoryBinding, error) {
	account, err := s.deps.Accounts.Get(ctx, req.AccountID)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}
	if err := account.Validate(); err != nil {
		return domain.RepositoryBinding{}, err
	}

	repo, err := s.deps.Git.DiscoverRepository(ctx, req.Path)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}

	remoteExplicitPort, err := validateRemote(account, repo)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}

	if existing, found, err := s.deps.Bindings.FindByRepository(ctx, repo.RootPath); err != nil {
		return domain.RepositoryBinding{}, err
	} else if found {
		return domain.RepositoryBinding{}, fmt.Errorf("%w: %s is bound to %s", domain.ErrAlreadyBound, repo.RootPath, existing.AccountID)
	}

	strategy, err := s.deps.Routing.Get(domain.StrategyRepoLocal)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}

	if snapshot, ok, err := s.deps.Snapshots.Load(repo.GitDir); err != nil {
		return domain.RepositoryBinding{}, err
	} else if ok && snapshot.BindingID != "" {
		return domain.RepositoryBinding{}, fmt.Errorf("%w: repository still carries gitra metadata", domain.ErrAlreadyBound)
	}

	bindingID, err := s.newBindingID()
	if err != nil {
		return domain.RepositoryBinding{}, err
	}

	entries, err := s.desiredEntries(ctx, account, bindingID, remoteExplicitPort)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}

	previous, err := s.capturePrevious(ctx, repo, entries)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}
	snapshot := ports.Snapshot{SchemaVersion: 1, BindingID: bindingID, Previous: previous}

	if err := strategy.Apply(ctx, s.deps.Git, repo, entries); err != nil {
		s.rollback(ctx, repo, strategy, previous, bindingID, false)
		return domain.RepositoryBinding{}, err
	}
	if err := s.deps.Snapshots.Save(repo.GitDir, snapshot); err != nil {
		s.rollback(ctx, repo, strategy, previous, bindingID, false)
		return domain.RepositoryBinding{}, err
	}

	binding := domain.RepositoryBinding{
		ID:         bindingID,
		AccountID:  account.ID,
		Repository: domain.RepositoryRef{Path: repo.RootPath},
		Strategy:   domain.StrategyRepoLocal,
		Revision:   1,
	}
	if err := s.deps.Bindings.Save(ctx, binding); err != nil {
		s.rollback(ctx, repo, strategy, previous, bindingID, false)
		return domain.RepositoryBinding{}, err
	}

	status, err := strategy.Inspect(ctx, s.deps.Git, repo, entries)
	if err != nil {
		s.rollback(ctx, repo, strategy, previous, bindingID, true)
		return domain.RepositoryBinding{}, err
	}
	if len(status.Drift) > 0 {
		s.rollback(ctx, repo, strategy, previous, bindingID, true)
		return domain.RepositoryBinding{}, fmt.Errorf("%w: verification failed for %v", domain.ErrStateDrift, status.Drift)
	}
	return binding, nil
}

// Unbind restores the pre-bind state and removes both records.
func (s *BindingService) Unbind(ctx context.Context, req UnbindRequest) error {
	return s.deps.Locker.WithWriteLock(ctx, func() error {
		repo, err := s.deps.Git.DiscoverRepository(ctx, req.Path)
		if err != nil {
			return err
		}
		strategy, err := s.deps.Routing.Get(domain.StrategyRepoLocal)
		if err != nil {
			return err
		}
		binding, found, err := s.deps.Bindings.FindByRepository(ctx, repo.RootPath)
		if err != nil {
			return err
		}
		snapshot, hasSnapshot, err := s.deps.Snapshots.Load(repo.GitDir)
		if err != nil {
			return err
		}
		if !found && !hasSnapshot {
			return fmt.Errorf("%w: %s", domain.ErrBindingNotFound, repo.RootPath)
		}

		if hasSnapshot {
			previous := toPreviousEntries(snapshot.Previous, managedKeys())
			if err := strategy.Restore(ctx, s.deps.Git, repo, previous); err != nil {
				return err
			}
			if err := s.deps.Snapshots.Delete(repo.GitDir); err != nil {
				return err
			}
		}
		// Keys added after the snapshot (or without one) are cleared explicitly.
		for _, key := range managedKeys() {
			if _, recorded := snapshot.Previous[key]; hasSnapshot && recorded {
				continue
			}
			if err := s.deps.Git.UnsetLocalConfig(ctx, repo, key); err != nil {
				return err
			}
		}
		if found {
			if err := s.deps.Bindings.Delete(ctx, binding.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// Status reports binding health for one repository path.
func (s *BindingService) Status(ctx context.Context, path string) (BindingStatus, error) {
	repo, err := s.deps.Git.DiscoverRepository(ctx, path)
	if err != nil {
		return BindingStatus{}, err
	}
	status := BindingStatus{Repository: repo.RootPath}

	binding, found, err := s.deps.Bindings.FindByRepository(ctx, repo.RootPath)
	if err != nil {
		return BindingStatus{}, err
	}
	bindingID, accountID, metadataPresent, err := s.readMetadata(ctx, repo)
	if err != nil {
		return BindingStatus{}, err
	}
	metadataMatches := metadataPresent && found && bindingID == string(binding.ID) && accountID == string(binding.AccountID)

	in := healthInput{
		HasCentralBinding: found,
		RepoExists:        true,
		MetadataPresent:   metadataPresent,
		MetadataMatches:   metadataMatches,
	}
	if found {
		status.Binding = binding
		account, err := s.deps.Accounts.Get(ctx, binding.AccountID)
		if err != nil {
			return BindingStatus{}, err
		}
		status.Account = account
		explicit := remoteExplicitPort(repo)
		entries, err := s.desiredEntries(ctx, account, binding.ID, explicit)
		if err != nil {
			return BindingStatus{}, err
		}
		if strategy, err := s.deps.Routing.Get(domain.StrategyRepoLocal); err == nil {
			if observed, err := strategy.Inspect(ctx, s.deps.Git, repo, entries); err == nil {
				in.Drift = observed.Drift
			}
		}
	}
	health, bound := computeHealth(in)
	status.Bound = bound
	status.Health = health
	status.Drift = in.Drift
	return status, nil
}

func (s *BindingService) desiredEntries(ctx context.Context, account domain.Account, bindingID domain.BindingID, remoteExplicitPort bool) ([]ports.GitConfigEntry, error) {
	return desiredEntries(ctx, s.deps, account, bindingID, remoteExplicitPort)
}

func desiredEntries(ctx context.Context, deps Deps, account domain.Account, bindingID domain.BindingID, remoteExplicitPort bool) ([]ports.GitConfigEntry, error) {
	strategy, err := deps.Auth.Get(account.Transport.Strategy)
	if err != nil {
		return nil, err
	}
	if err := strategy.Validate(ctx, account); err != nil {
		return nil, err
	}
	authEntries, err := strategy.BuildGitConfig(auth.BuildRequest{Account: account, RemoteHasExplicitPort: remoteExplicitPort})
	if err != nil {
		return nil, err
	}
	entries := []ports.GitConfigEntry{
		{Key: "user.name", Value: account.Identity.Name},
		{Key: "user.email", Value: account.Identity.Email},
	}
	entries = append(entries, authEntries...)
	entries = append(entries,
		ports.GitConfigEntry{Key: "gitra.bindingId", Value: string(bindingID)},
		ports.GitConfigEntry{Key: "gitra.accountId", Value: string(account.ID)},
		ports.GitConfigEntry{Key: "gitra.strategy", Value: domain.StrategyRepoLocal},
		ports.GitConfigEntry{Key: "gitra.version", Value: "1"},
	)
	return entries, nil
}

func (s *BindingService) capturePrevious(ctx context.Context, repo domain.Repository, entries []ports.GitConfigEntry) (map[string]ports.ConfigState, error) {
	previous := make(map[string]ports.ConfigState, len(entries))
	for _, entry := range entries {
		value, found, err := s.deps.Git.GetLocalConfig(ctx, repo, entry.Key)
		if err != nil {
			return nil, err
		}
		previous[entry.Key] = ports.ConfigState{Exists: found, Value: value}
	}
	return previous, nil
}

// rollback restores the snapshot and removes partial metadata/binding.
func (s *BindingService) rollback(ctx context.Context, repo domain.Repository, strategy routing.Strategy, previous map[string]ports.ConfigState, bindingID domain.BindingID, centralSaved bool) {
	_ = strategy.Restore(ctx, s.deps.Git, repo, toPreviousEntries(previous, managedKeys()))
	_ = s.deps.Snapshots.Delete(repo.GitDir)
	if centralSaved {
		_ = s.deps.Bindings.Delete(ctx, bindingID)
	}
}

func toPreviousEntries(previous map[string]ports.ConfigState, keys []string) []routing.PreviousEntry {
	entries := make([]routing.PreviousEntry, 0, len(keys))
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	for _, key := range sorted {
		state, ok := previous[key]
		if !ok {
			entries = append(entries, routing.PreviousEntry{Key: key, Exists: false})
			continue
		}
		entries = append(entries, routing.PreviousEntry{Key: key, Exists: state.Exists, Value: state.Value})
	}
	return entries
}

func (s *BindingService) readMetadata(ctx context.Context, repo domain.Repository) (string, string, bool, error) {
	return readRepoMetadata(ctx, s.deps, repo)
}

func readRepoMetadata(ctx context.Context, deps Deps, repo domain.Repository) (string, string, bool, error) {
	bindingID, hasBindingID, err := deps.Git.GetLocalConfig(ctx, repo, "gitra.bindingId")
	if err != nil {
		return "", "", false, err
	}
	accountID, hasAccountID, err := deps.Git.GetLocalConfig(ctx, repo, "gitra.accountId")
	if err != nil {
		return "", "", false, err
	}
	if !hasBindingID || !hasAccountID || bindingID == "" || accountID == "" {
		return "", "", false, nil
	}
	return bindingID, accountID, true, nil
}

// validateRemote enforces SSH-only transport and provider/host compatibility
// (baseline §4.2, §4.7). It returns whether the URL carries its own port.
func validateRemote(account domain.Account, repo domain.Repository) (bool, error) {
	remote, ok := repo.RemoteByName("origin")
	if !ok {
		if len(repo.Remotes) == 0 {
			return false, nil
		}
		remote = repo.Remotes[0]
	}
	parsed, err := domain.ParseRemoteURL(remote.URL)
	if err != nil {
		return false, fmt.Errorf("%w (%s): %v", domain.ErrUnsupportedRemote, remote.Name, err)
	}
	if !equalFoldHost(parsed.Host, account.Provider.Endpoint.Host) {
		return false, fmt.Errorf("%w: remote host %q does not match account endpoint %q", domain.ErrProviderMismatch, parsed.Host, account.Provider.Endpoint.Host)
	}
	return parsed.ExplicitPort, nil
}

func remoteExplicitPort(repo domain.Repository) bool {
	remote, ok := repo.RemoteByName("origin")
	if !ok {
		if len(repo.Remotes) == 0 {
			return false
		}
		remote = repo.Remotes[0]
	}
	parsed, err := domain.ParseRemoteURL(remote.URL)
	if err != nil {
		return false
	}
	return parsed.ExplicitPort
}

func equalFoldHost(a, b string) bool {
	return len(a) == len(b) && (a == b || stringFold(a) == stringFold(b))
}

func stringFold(s string) string {
	buf := []byte(s)
	for i := range buf {
		if buf[i] >= 'A' && buf[i] <= 'Z' {
			buf[i] += 'a' - 'A'
		}
	}
	return string(buf)
}
