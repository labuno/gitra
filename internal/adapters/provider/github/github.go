// Package github implements GitHub token reuse and profile lookup.
package github

import (
	"context"
	"net/http"

	"github.com/zhanhd/gitra/internal/adapters/provider"
	"github.com/zhanhd/gitra/internal/domain"
)

// DefaultBaseURL is the public GitHub API endpoint.
const DefaultBaseURL = "https://api.github.com"

// Client talks to the GitHub REST API.
type Client struct {
	BaseURL string
	HTTP    provider.HTTPDoer
}

// New builds a client; empty baseURL falls back to the public API.
func New(baseURL string, httpClient provider.HTTPDoer) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{BaseURL: baseURL, HTTP: httpClient}
}

type userPayload struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type emailPayload struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// Profile returns the authenticated account profile for host.
func (c *Client) Profile(ctx context.Context, token, host string) (provider.Profile, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}
	var user userPayload
	if err := provider.DoJSON(ctx, c.HTTP, http.MethodGet, c.BaseURL+"/user", headers, &user); err != nil {
		return provider.Profile{}, err
	}

	var emails []emailPayload
	_ = provider.DoJSON(ctx, c.HTTP, http.MethodGet, c.BaseURL+"/user/emails", headers, &emails)

	profile := provider.Profile{Type: domain.ProviderGitHub, Host: host, Username: user.Login, Name: user.Name}
	for _, email := range emails {
		if email.Primary && email.Verified {
			profile.Email = email.Email
			break
		}
	}
	if profile.Email == "" && len(emails) > 0 {
		profile.Email = emails[0].Email
	}
	if profile.Email == "" {
		profile.Email = user.Login + "@users.noreply.github.com"
		profile.EmailFallback = true
	}
	if profile.Name == "" {
		profile.Name = user.Login
	}
	return profile, nil
}
