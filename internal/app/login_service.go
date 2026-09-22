package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// loginTimeout bounds one complete login attempt.
const loginTimeout = 45 * time.Second

// LoginRequest describes one provider login.
type LoginRequest struct {
	Provider      domain.ProviderType
	Host          string
	Alias         string
	Token         string
	AllowStdin    bool
	AllowCLIReuse bool
}

// LoginResult is the outcome of a successful login.
type LoginResult struct {
	Account     domain.Account
	Profile     ports.ProviderProfile
	Created     bool
	TokenSource string
}

// LoginService orchestrates provider login: acquire a credential, store it,
// then create or update the https-token account (addendum §6).
type LoginService struct {
	deps     Deps
	accounts *AccountService
	tokens   ports.TokenProvider
	profiles ports.ProfileProvider
}

// NewLoginService wires the login use case.
func NewLoginService(deps Deps, accounts *AccountService, tokens ports.TokenProvider, profiles ports.ProfileProvider) *LoginService {
	return &LoginService{deps: deps, accounts: accounts, tokens: tokens, profiles: profiles}
}

// Login implements the provider login flow. The token is stored before the
// account is written so account validation can see it.
func (s *LoginService) Login(ctx context.Context, req LoginRequest) (LoginResult, error) {
	if s.tokens == nil || s.profiles == nil || s.deps.Secrets == nil {
		return LoginResult{}, errors.New("login is not available in this build")
	}
	// Bound the whole login: acquisition may probe a CLI, profile lookup goes
	// over the network, and storage may wait on a Keychain prompt.
	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()
	provider, err := s.providerEndpoint(req)
	if err != nil {
		return LoginResult{}, err
	}

	token, err := s.tokens.Acquire(ctx, req.Provider, req.Token, req.AllowStdin, req.AllowCLIReuse)
	if err != nil {
		return LoginResult{}, err
	}
	profile, err := s.profiles.Profile(ctx, req.Provider, provider.Host, token.Value)
	if err != nil {
		return LoginResult{}, err
	}

	alias := req.Alias
	if alias == "" {
		alias = profile.Username
	}

	existing, found, err := s.findAccount(ctx, req.Provider, provider.Host, profile.Username)
	if err != nil {
		return LoginResult{}, err
	}

	ref := provider.Host + "/" + profile.Username
	if err := s.deps.Secrets.Set(ctx, ref, token.Value); err != nil {
		return LoginResult{}, err
	}

	if found {
		next := existing
		if req.Alias != "" {
			next.Alias = alias
		}
		next.Provider.Endpoint = provider
		next.Provider.Username = profile.Username
		if next.Identity.Name == "" {
			next.Identity.Name = profile.Name
		}
		if next.Identity.Email == "" {
			next.Identity.Email = profile.Email
		}
		updated, err := s.accounts.Update(ctx, UpdateAccountRequest{Account: next})
		if err != nil {
			return LoginResult{Account: updated, Profile: profile, Created: false, TokenSource: token.Source}, err
		}
		return LoginResult{Account: updated, Profile: profile, Created: false, TokenSource: token.Source}, nil
	}

	account, err := s.accounts.Create(ctx, CreateAccountRequest{
		Alias: alias,
		Provider: domain.ProviderRef{
			Type:     req.Provider,
			Username: profile.Username,
			Endpoint: provider,
		},
		Identity:  domain.CommitIdentity{Name: profile.Name, Email: profile.Email},
		Transport: domain.TransportConfig{Strategy: domain.StrategyHTTPSToken, Config: map[string]string{}},
	})
	if err != nil {
		_ = s.deps.Secrets.Delete(ctx, ref)
		return LoginResult{}, err
	}
	return LoginResult{Account: account, Profile: profile, Created: true, TokenSource: token.Source}, nil
}

// Logout deletes the stored credential and clears the repo-local credential
// scope of every bound repository. The account itself stays, so status can
// report needs_login until the user logs in again.
func (s *LoginService) Logout(ctx context.Context, accountID domain.AccountID) error {
	account, err := s.deps.Accounts.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Transport.Strategy != domain.StrategyHTTPSToken {
		return fmt.Errorf("%w: account %s does not use a stored provider credential", domain.ErrInvalid, account.Alias)
	}
	if err := s.deps.Secrets.Delete(ctx, account.CredentialRef()); err != nil {
		return err
	}
	bindings, err := s.deps.Bindings.ListByAccount(ctx, accountID)
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		repo, err := s.deps.Git.DiscoverRepository(ctx, binding.Repository.Path)
		if err != nil {
			continue // moved or deleted repository: nothing to clean
		}
		for _, key := range credentialScopeKeys(account) {
			if err := s.deps.Git.UnsetLocalConfig(ctx, repo, key); err != nil {
				return err
			}
		}
	}
	return nil
}

// credentialScopeKeys are the transport-specific managed keys of an
// https-token account (identity and gitra.* metadata stay in place).
func credentialScopeKeys(account domain.Account) []string {
	return []string{
		"credential.https://" + account.Provider.Endpoint.Host + ".username",
		"credential.helper",
	}
}

func (s *LoginService) providerEndpoint(req LoginRequest) (domain.ProviderEndpoint, error) {
	endpoint, ok := domain.DefaultEndpoint(req.Provider)
	if !ok {
		return domain.ProviderEndpoint{}, fmt.Errorf("%w: unknown provider %q", domain.ErrInvalid, req.Provider)
	}
	if req.Host != "" {
		endpoint.Host = req.Host
	}
	return endpoint, nil
}

func (s *LoginService) findAccount(ctx context.Context, providerType domain.ProviderType, host, username string) (domain.Account, bool, error) {
	accounts, err := s.deps.Accounts.List(ctx)
	if err != nil {
		return domain.Account{}, false, err
	}
	for _, account := range accounts {
		if account.Provider.Type == providerType && account.Provider.Endpoint.Host == host && account.Provider.Username == username {
			return account, true, nil
		}
	}
	return domain.Account{}, false, nil
}
