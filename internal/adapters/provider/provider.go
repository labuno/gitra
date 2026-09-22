// Package provider implements provider-side login helpers: token acquisition
// and profile lookup. Device flow is intentionally not implemented yet; it
// waits for a registered OAuth client id (addendum §4.5).
package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// ErrProfileUnavailable marks provider failures that are not credential
// problems: network errors, 5xx responses, malformed payloads.
var ErrProfileUnavailable = errors.New("provider profile unavailable")

// cliReuseTimeout bounds optional gh/glab probing.
const cliReuseTimeout = 5 * time.Second

// Profile is the provider account information used to auto-fill an account.
type Profile struct {
	Type          domain.ProviderType
	Host          string
	Username      string
	Name          string
	Email         string
	EmailFallback bool
}

// Token is one acquired credential plus where it came from.
type Token struct {
	Value  string
	Source string // explicit | env | stdin | gh | glab
}

// TokenRequest describes which acquisition paths are allowed.
type TokenRequest struct {
	Provider      domain.ProviderType
	ExplicitToken string
	AllowStdin    bool
	AllowCLIReuse bool
}

// TokenResolver implements the login ladder: explicit → env → stdin → CLI.
type TokenResolver struct {
	runner ports.CommandRunner
	stdin  io.Reader
	Getenv func(string) string
}

// NewTokenResolver builds the resolver.
func NewTokenResolver(runner ports.CommandRunner, stdin io.Reader) *TokenResolver {
	return &TokenResolver{runner: runner, stdin: stdin, Getenv: os.Getenv}
}

// Resolve returns one usable token or an actionable ErrAuthInvalid.
func (r *TokenResolver) Resolve(ctx context.Context, req TokenRequest) (Token, error) {
	if value := strings.TrimSpace(req.ExplicitToken); value != "" {
		return Token{Value: value, Source: "explicit"}, nil
	}
	if r.Getenv != nil {
		if value := strings.TrimSpace(r.Getenv("GITRA_TOKEN")); value != "" {
			return Token{Value: value, Source: "env"}, nil
		}
	}
	if req.AllowStdin && r.stdin != nil {
		data, err := io.ReadAll(io.LimitReader(r.stdin, 8192))
		if err != nil {
			return Token{}, fmt.Errorf("read token from stdin: %w", err)
		}
		if value := strings.TrimSpace(string(data)); value != "" {
			return Token{Value: value, Source: "stdin"}, nil
		}
	}
	if req.AllowCLIReuse {
		if token, err := r.reuseCLI(ctx, req.Provider); err == nil && token.Value != "" {
			return token, nil
		}
	}
	return Token{}, fmt.Errorf(
		"%w: no credential available for %s.\nProvide one with:\n  gitra login --provider %s --token-stdin   (paste a personal access token)\nor export GITRA_TOKEN.\n%s",
		domain.ErrAuthInvalid, req.Provider, req.Provider, guidance(req.Provider))
}

func (r *TokenResolver) reuseCLI(ctx context.Context, providerType domain.ProviderType) (Token, error) {
	// gh/glab can hang (network, locked keychain), and they are a convenience
	// path only: bound the probe so the UI never freezes.
	ctx, cancel := context.WithTimeout(ctx, cliReuseTimeout)
	defer cancel()
	switch providerType {
	case domain.ProviderGitHub:
		result, err := r.runner.Run(ctx, "gh", "auth", "token")
		if err != nil || result.ExitCode != 0 {
			return Token{}, errors.New("gh CLI has no usable session")
		}
		return Token{Value: strings.TrimSpace(result.Stdout), Source: "gh"}, nil
	case domain.ProviderGitLab:
		result, err := r.runner.Run(ctx, "glab", "auth", "status", "--show-token")
		if err != nil || result.ExitCode != 0 {
			return Token{}, errors.New("glab CLI has no usable session")
		}
		for _, line := range strings.Split(result.Stdout+result.Stderr, "\n") {
			if value, ok := strings.CutPrefix(strings.TrimSpace(line), "Token:"); ok {
				if token := strings.TrimSpace(value); token != "" {
					return Token{Value: token, Source: "glab"}, nil
				}
			}
		}
	}
	return Token{}, errors.New("no reusable CLI session")
}

func guidance(providerType domain.ProviderType) string {
	switch providerType {
	case domain.ProviderGitHub:
		return "Create a token at https://github.com/settings/tokens with scopes: repo, read:user, user:email."
	case domain.ProviderGitLab:
		return "Create a token at https://gitlab.com/-/user_settings/personal_access_tokens with scopes: api, read_user."
	default:
		return "Create a token in your Gitea instance under Settings → Applications."
	}
}

// HTTPDoer is satisfied by *http.Client.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// DoJSON performs one JSON request, classifying credential vs availability
// failures so callers can map them to the right exit code.
func DoJSON(ctx context.Context, client HTTPDoer, method, url string, headers map[string]string, out any) error {
	request, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProfileUnavailable, err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProfileUnavailable, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProfileUnavailable, err)
	}
	switch {
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: provider rejected the token (HTTP %d)", domain.ErrAuthInvalid, response.StatusCode)
	case response.StatusCode >= 300:
		return fmt.Errorf("%w: provider returned HTTP %d", ErrProfileUnavailable, response.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: invalid response payload: %v", ErrProfileUnavailable, err)
	}
	return nil
}
