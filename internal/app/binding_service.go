// Package app contains application services. This file owns the binding
// lifecycle: snapshot, apply, verify, rollback, unbind and status.
package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

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

	// Secrets and CredentialHelpers serve the https-token strategy (V1.1).
	Secrets           ports.SecretStore
	CredentialHelpers ports.CredentialHelperResolver
	// SSH serves explicit account verification for ssh-key accounts.
	SSH ports.SSH
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
	// NeedsLogin is true when the account credential is missing (e.g. after
	// logout) while repository metadata is still present.
	NeedsLogin bool
	// HasOrigin reports whether the repository already has an "origin" remote,
	// and OriginURL carries its address. gitra never rewrites an existing
	// remote (baseline §7); it may only offer to add a missing one.
	HasOrigin bool
	OriginURL string
}

// managedKeyNames is the ownership whitelist for one account (baseline §9):
// identity keys plus the transport-specific entries for its strategy.
func managedKeyNames(account domain.Account) []string {
	keys := []string{
		"user.name",
		"user.email",
		"gitra.bindingId",
		"gitra.accountId",
		"gitra.strategy",
		"gitra.version",
	}
	switch account.Transport.Strategy {
	case domain.StrategyHTTPSToken:
		keys = append(keys,
			"credential.https://"+account.Provider.Endpoint.Host+".username",
			"credential.helper",
		)
	default:
		keys = append(keys, "core.sshCommand")
	}
	sort.Strings(keys)
	return keys
}

// DesiredKeys exposes the managed key names for a strategy (used by Unbind
// fallbacks and tests).
func DesiredKeys(account domain.Account) []string { return managedKeyNames(account) }

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

	authStrategy, err := s.deps.Auth.Get(account.Transport.Strategy)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}
	remoteHost, remoteExplicitPort, err := validateRemote(account, repo, authStrategy.Transport())
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

	entries, err := s.desiredEntries(ctx, account, bindingID, remoteHost, remoteExplicitPort)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}

	previous, err := s.capturePrevious(ctx, repo, entries)
	if err != nil {
		return domain.RepositoryBinding{}, err
	}
	snapshot := ports.Snapshot{SchemaVersion: 1, BindingID: bindingID, Previous: previous}

	if err := strategy.Apply(ctx, s.deps.Git, repo, entries); err != nil {
		s.rollback(ctx, repo, strategy, previous, managedKeyNames(account), bindingID, false)
		return domain.RepositoryBinding{}, err
	}
	if err := s.deps.Snapshots.Save(repo.GitDir, snapshot); err != nil {
		s.rollback(ctx, repo, strategy, previous, managedKeyNames(account), bindingID, false)
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
		s.rollback(ctx, repo, strategy, previous, managedKeyNames(account), bindingID, false)
		return domain.RepositoryBinding{}, err
	}

	status, err := strategy.Inspect(ctx, s.deps.Git, repo, entries)
	if err != nil {
		s.rollback(ctx, repo, strategy, previous, managedKeyNames(account), bindingID, true)
		return domain.RepositoryBinding{}, err
	}
	if len(status.Drift) > 0 {
		s.rollback(ctx, repo, strategy, previous, managedKeyNames(account), bindingID, true)
		return domain.RepositoryBinding{}, fmt.Errorf("%w: verification failed for %v", domain.ErrStateDrift, status.Drift)
	}
	return binding, nil
}

// EnsureOriginRemote adds an "origin" remote when the repository has none.
// An existing origin is never touched (baseline §7).
func (s *BindingService) EnsureOriginRemote(ctx context.Context, path, url string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("%w: repository address must not be empty", domain.ErrInvalid)
	}
	parsed, err := domain.ParseHTTPSRemoteURL(url)
	if err != nil {
		if _, sshErr := domain.ParseRemoteURL(url); sshErr != nil {
			return fmt.Errorf("%w: %s", domain.ErrUnsupportedRemote, url)
		}
	} else if parsed.Host == "" {
		return fmt.Errorf("%w: %s", domain.ErrUnsupportedRemote, url)
	}

	return s.deps.Locker.WithWriteLock(ctx, func() error {
		repo, err := s.deps.Git.DiscoverRepository(ctx, path)
		if err != nil {
			return err
		}
		if _, ok := repo.RemoteByName("origin"); ok {
			return nil // never rewrite an existing remote
		}
		if err := s.deps.Git.SetLocalConfig(ctx, repo, "remote.origin.url", strings.TrimSpace(url)); err != nil {
			return err
		}
		return s.deps.Git.SetLocalConfig(ctx, repo, "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	})
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
		var account domain.Account
		if found {
			account, err = s.deps.Accounts.Get(ctx, binding.AccountID)
			if err != nil {
				return err
			}
		}
		keys := managedKeyNames(account)

		if hasSnapshot {
			previous := toPreviousEntries(snapshot.Previous, keys)
			if err := strategy.Restore(ctx, s.deps.Git, repo, previous); err != nil {
				return err
			}
			if err := s.deps.Snapshots.Delete(repo.GitDir); err != nil {
				return err
			}
		}
		// Keys added after the snapshot (or without one) are cleared explicitly.
		for _, key := range keys {
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
	if origin, ok := repo.RemoteByName("origin"); ok {
		status.HasOrigin = true
		status.OriginURL = origin.URL
	}
	if found {
		status.Binding = binding
		account, err := s.deps.Accounts.Get(ctx, binding.AccountID)
		if err != nil {
			return BindingStatus{}, err
		}
		status.Account = account
		explicit := remoteExplicitPort(repo)
		entries, err := s.desiredEntries(ctx, account, binding.ID, account.Provider.Endpoint.Host, explicit)
		if errors.Is(err, domain.ErrAuthInvalid) {
			// Credential missing (for example after logout): keep reporting the
			// binding and surface needs_login instead of failing the command.
			status.Bound = true
			status.NeedsLogin = true
			status.Health = domain.HealthBroken
			return status, nil
		}
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

func (s *BindingService) desiredEntries(ctx context.Context, account domain.Account, bindingID domain.BindingID, remoteHost string, remoteExplicitPort bool) ([]ports.GitConfigEntry, error) {
	return desiredEntries(ctx, s.deps, account, bindingID, remoteHost, remoteExplicitPort)
}

func desiredEntries(ctx context.Context, deps Deps, account domain.Account, bindingID domain.BindingID, remoteHost string, remoteExplicitPort bool) ([]ports.GitConfigEntry, error) {
	strategy, err := deps.Auth.Get(account.Transport.Strategy)
	if err != nil {
		return nil, err
	}
	if err := strategy.Validate(ctx, account); err != nil {
		return nil, err
	}
	helper := ""
	if deps.CredentialHelpers != nil {
		spec, ok, err := deps.CredentialHelpers.Helper(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			helper = spec
		}
	}
	authEntries, err := strategy.BuildGitConfig(auth.BuildRequest{
		Account: account, RemoteHost: remoteHost,
		RemoteHasExplicitPort: remoteExplicitPort, CredentialHelper: helper,
	})
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
func (s *BindingService) rollback(ctx context.Context, repo domain.Repository, strategy routing.Strategy, previous map[string]ports.ConfigState, keys []string, bindingID domain.BindingID, centralSaved bool) {
	_ = strategy.Restore(ctx, s.deps.Git, repo, toPreviousEntries(previous, keys))
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

// validateRemote enforces that the remote transport matches the account auth
// strategy and that the host matches the account endpoint (baseline §4.2/§4.7,
// addendum §5.3). It returns the remote host and whether the URL carries a port.
func validateRemote(account domain.Account, repo domain.Repository, transport domain.RemoteTransport) (string, bool, error) {
	remote, ok := repo.RemoteByName("origin")
	if !ok {
		if len(repo.Remotes) == 0 {
			return account.Provider.Endpoint.Host, false, nil
		}
		remote = repo.Remotes[0]
	}
	var (
		parsed domain.RemoteURL
		err    error
	)
	if transport == domain.RemoteTransportHTTPS {
		parsed, err = domain.ParseHTTPSRemoteURL(remote.URL)
	} else {
		parsed, err = domain.ParseRemoteURL(remote.URL)
	}
	if err != nil {
		return "", false, fmt.Errorf("%w (%s): %v", domain.ErrUnsupportedRemote, remote.Name, err)
	}
	if !equalFoldHost(parsed.Host, account.Provider.Endpoint.Host) {
		return "", false, fmt.Errorf("%w: remote host %q does not match account endpoint %q", domain.ErrProviderMismatch, parsed.Host, account.Provider.Endpoint.Host)
	}
	return parsed.Host, parsed.ExplicitPort, nil
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
