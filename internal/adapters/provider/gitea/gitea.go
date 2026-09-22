// Package gitea implements Gitea/Forgejo profile lookup (token based in V1.1).
package gitea

import (
	"context"
	"net/http"

	"github.com/zhanhd/gitra/internal/adapters/provider"
	"github.com/zhanhd/gitra/internal/domain"
)

// Client talks to a Gitea instance REST API.
type Client struct {
	BaseURL string
	HTTP    provider.HTTPDoer
}

// New builds a client; baseURL should look like https://gitea.example.com.
func New(baseURL string, httpClient provider.HTTPDoer) *Client {
	if baseURL == "" {
		baseURL = "https://gitea.com"
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{BaseURL: baseURL, HTTP: httpClient}
}

type userPayload struct {
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

// Profile returns the authenticated account profile for host.
func (c *Client) Profile(ctx context.Context, token, host string) (provider.Profile, error) {
	headers := map[string]string{"Authorization": "token " + token}
	var user userPayload
	if err := provider.DoJSON(ctx, c.HTTP, http.MethodGet, c.BaseURL+"/api/v1/user", headers, &user); err != nil {
		return provider.Profile{}, err
	}
	profile := provider.Profile{Type: domain.ProviderGitea, Host: host, Username: user.Login, Name: user.FullName, Email: user.Email}
	if profile.Email == "" {
		profile.Email = user.Login + "@" + host
		profile.EmailFallback = true
	}
	if profile.Name == "" {
		profile.Name = user.Login
	}
	return profile, nil
}
