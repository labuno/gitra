package github

import (
	"context"
	"errors"
	"net/http"

	"github.com/zhanhd/gitra/internal/adapters/provider"
	"github.com/zhanhd/gitra/internal/ports"
)

type repoPayload struct {
	Private  bool   `json:"private"`
	CloneURL string `json:"clone_url"`
	Name     string `json:"name"`
}

// LookupRepo reports whether owner/name exists and is visible to the token.
func (c *Client) LookupRepo(ctx context.Context, token, owner, name string) (ports.RemoteStatus, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}
	var payload repoPayload
	err := provider.DoJSON(ctx, c.HTTP, http.MethodGet, c.BaseURL+"/repos/"+owner+"/"+name, headers, &payload)
	if err != nil {
		if errors.Is(err, provider.ErrNotFound) {
			return ports.RemoteStatus{Exists: false}, nil
		}
		return ports.RemoteStatus{}, err
	}
	return ports.RemoteStatus{Exists: true, Private: payload.Private, CloneURL: payload.CloneURL}, nil
}

// CreateRepo creates a repository under the authenticated user.
func (c *Client) CreateRepo(ctx context.Context, token, owner, name string, private bool) (ports.RemoteStatus, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}
	body := map[string]any{"name": name, "private": private, "auto_init": false}
	var payload repoPayload
	err := provider.DoJSONBody(ctx, c.HTTP, http.MethodPost, c.BaseURL+"/user/repos", headers, body, &payload)
	if err != nil {
		return ports.RemoteStatus{}, err
	}
	return ports.RemoteStatus{Exists: true, Private: payload.Private, CloneURL: payload.CloneURL}, nil
}
