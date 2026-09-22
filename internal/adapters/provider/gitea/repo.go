package gitea

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
}

// LookupRepo reports whether owner/name exists for this token.
func (c *Client) LookupRepo(ctx context.Context, token, owner, name string) (ports.RemoteStatus, error) {
	headers := map[string]string{"Authorization": "token " + token}
	var payload repoPayload
	err := provider.DoJSON(ctx, c.HTTP, http.MethodGet, c.BaseURL+"/api/v1/repos/"+owner+"/"+name, headers, &payload)
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
	headers := map[string]string{"Authorization": "token " + token}
	body := map[string]any{"name": name, "private": private}
	var payload repoPayload
	if err := provider.DoJSONBody(ctx, c.HTTP, http.MethodPost, c.BaseURL+"/api/v1/user/repos", headers, body, &payload); err != nil {
		return ports.RemoteStatus{}, err
	}
	return ports.RemoteStatus{Exists: true, Private: private, CloneURL: payload.CloneURL}, nil
}
