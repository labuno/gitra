// Package router routes provider login/profile calls to the concrete provider
// adapters. It is composition-layer code: the mapping from provider type to
// client implementation lives here, never in the business layer.
package router

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/zhanhd/gitra/internal/adapters/provider"
	"github.com/zhanhd/gitra/internal/adapters/provider/gitea"
	"github.com/zhanhd/gitra/internal/adapters/provider/github"
	"github.com/zhanhd/gitra/internal/adapters/provider/gitlab"
	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

var (
	_ ports.TokenProvider     = (*Adapter)(nil)
	_ ports.ProfileProvider   = (*Adapter)(nil)
	_ ports.SSHIdentityParser = (*Adapter)(nil)
	_ ports.RepoService       = (*Adapter)(nil)
)

// Adapter implements ports.TokenProvider and ports.ProfileProvider.
type Adapter struct {
	resolver *provider.TokenResolver
	http     provider.HTTPDoer
	// BaseURL overrides provider endpoints (tests); nil uses host-derived URLs.
	BaseURL func(providerType domain.ProviderType, host string) string
}

// DefaultHTTPTimeout bounds every provider API call so a stalled network can
// never freeze the interface.
const DefaultHTTPTimeout = 20 * time.Second

// New builds the adapter over a process runner and stdin.
func New(runner ports.CommandRunner, stdin io.Reader) *Adapter {
	return &Adapter{
		resolver: provider.NewTokenResolver(runner, stdin),
		http:     &http.Client{Timeout: DefaultHTTPTimeout},
	}
}

// defaultBaseURL derives API endpoints from the provider and host.
func defaultBaseURL(providerType domain.ProviderType, host string) string {
	switch providerType {
	case domain.ProviderGitHub:
		if host == "" || host == "github.com" {
			return "https://api.github.com"
		}
		return "https://" + host + "/api/v3"
	case domain.ProviderGitLab:
		return "https://" + host
	case domain.ProviderGitea:
		return "https://" + host
	default:
		return ""
	}
}

func (a *Adapter) baseURL(providerType domain.ProviderType, host string) string {
	if a.BaseURL != nil {
		return a.BaseURL(providerType, host)
	}
	return defaultBaseURL(providerType, host)
}

// Acquire implements ports.TokenProvider.
func (a *Adapter) Acquire(ctx context.Context, req ports.TokenRequest) (ports.LoginToken, error) {
	token, err := a.resolver.Resolve(ctx, provider.TokenRequest{
		Provider:      req.Provider,
		Username:      req.Username,
		ExplicitToken: req.Explicit,
		AllowStdin:    req.AllowStdin,
		AllowCLIReuse: req.AllowCLIReuse,
	})
	if err != nil {
		return ports.LoginToken{}, err
	}
	return ports.LoginToken{Value: token.Value, Source: token.Source}, nil
}

// Profile implements ports.ProfileProvider.
// httpClient returns the adapter's client, creating a bounded one on demand.
func (a *Adapter) httpClient() provider.HTTPDoer {
	if a.http == nil {
		a.http = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	return a.http
}

func (a *Adapter) Profile(ctx context.Context, providerType domain.ProviderType, host, token string) (ports.ProviderProfile, error) {
	base := a.baseURL(providerType, host)
	var (
		profile provider.Profile
		err     error
	)
	switch providerType {
	case domain.ProviderGitHub:
		profile, err = github.New(base, a.httpClient()).Profile(ctx, token, host)
	case domain.ProviderGitLab:
		profile, err = gitlab.New(base, a.httpClient()).Profile(ctx, token, host)
	case domain.ProviderGitea:
		profile, err = gitea.New(base, a.httpClient()).Profile(ctx, token, host)
	default:
		return ports.ProviderProfile{}, domain.ErrProviderMismatch
	}
	if err != nil {
		return ports.ProviderProfile{}, err
	}
	return ports.ProviderProfile{
		Provider:      profile.Type,
		Host:          profile.Host,
		Username:      profile.Username,
		Name:          profile.Name,
		Email:         profile.Email,
		EmailFallback: profile.EmailFallback,
	}, nil
}

// ParseSSHIdentity implements ports.SSHIdentityParser.
func (a *Adapter) ParseSSHIdentity(_ context.Context, providerType domain.ProviderType, stdout, stderr string) (string, bool) {
	switch providerType {
	case domain.ProviderGitHub:
		return github.ParseSSHVerification(stdout, stderr)
	case domain.ProviderGitLab:
		return gitlab.ParseSSHVerification(stdout, stderr)
	case domain.ProviderGitea:
		return gitea.ParseSSHVerification(stdout, stderr)
	default:
		return "", false
	}
}

// LookupRepo implements ports.RepoService.
func (a *Adapter) LookupRepo(ctx context.Context, providerType domain.ProviderType, host, token, owner, repo string) (ports.RemoteStatus, error) {
	switch providerType {
	case domain.ProviderGitHub:
		return github.New(a.baseURL(providerType, host), a.httpClient()).LookupRepo(ctx, token, owner, repo)
	case domain.ProviderGitLab:
		return gitlab.New(a.baseURL(providerType, host), a.httpClient()).LookupRepo(ctx, token, owner, repo)
	case domain.ProviderGitea:
		return gitea.New(a.baseURL(providerType, host), a.httpClient()).LookupRepo(ctx, token, owner, repo)
	default:
		return ports.RemoteStatus{}, domain.ErrProviderMismatch
	}
}

// CreateRepo implements ports.RepoService.
func (a *Adapter) CreateRepo(ctx context.Context, providerType domain.ProviderType, host, token, owner, name string, private bool) (ports.RemoteStatus, error) {
	switch providerType {
	case domain.ProviderGitHub:
		return github.New(a.baseURL(providerType, host), a.httpClient()).CreateRepo(ctx, token, owner, name, private)
	case domain.ProviderGitLab:
		return gitlab.New(a.baseURL(providerType, host), a.httpClient()).CreateRepo(ctx, token, owner, name, private)
	case domain.ProviderGitea:
		return gitea.New(a.baseURL(providerType, host), a.httpClient()).CreateRepo(ctx, token, owner, name, private)
	default:
		return ports.RemoteStatus{}, domain.ErrProviderMismatch
	}
}
