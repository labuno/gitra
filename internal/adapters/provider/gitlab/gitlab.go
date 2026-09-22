// Package gitlab implements GitLab profile lookup (PAT based in V1.1).
package gitlab

import (
	"context"
	"net/http"

	"github.com/zhanhd/gitra/internal/adapters/provider"
	"github.com/zhanhd/gitra/internal/domain"
)

// BaseURLFor returns the API base URL for a GitLab host.
func BaseURLFor(host string) string { return "https://" + host }

// Client talks to the GitLab REST API.
type Client struct {
	BaseURL string
	HTTP    provider.HTTPDoer
}

// New builds a client; baseURL should look like https://gitlab.com.
func New(baseURL string, httpClient provider.HTTPDoer) *Client {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{BaseURL: baseURL, HTTP: httpClient}
}

type userPayload struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// Profile returns the authenticated account profile for host.
func (c *Client) Profile(ctx context.Context, token, host string) (provider.Profile, error) {
	headers := map[string]string{"PRIVATE-TOKEN": token}
	var user userPayload
	if err := provider.DoJSON(ctx, c.HTTP, http.MethodGet, c.BaseURL+"/api/v4/user", headers, &user); err != nil {
		return provider.Profile{}, err
	}
	profile := provider.Profile{Type: domain.ProviderGitLab, Host: host, Username: user.Username, Name: user.Name, Email: user.Email}
	if profile.Email == "" {
		profile.Email = user.Username + "@users.noreply.gitlab.com"
		profile.EmailFallback = true
	}
	if profile.Name == "" {
		profile.Name = user.Username
	}
	return profile, nil
}
