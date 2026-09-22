package gitlab

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/zhanhd/gitra/internal/adapters/provider"
	"github.com/zhanhd/gitra/internal/ports"
)

type projectPayload struct {
	PathWithNamespace string `json:"path_with_namespace"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	Visibility        string `json:"visibility"`
}

// LookupRepo reports whether owner/name exists for this token.
func (c *Client) LookupRepo(ctx context.Context, token, owner, name string) (ports.RemoteStatus, error) {
	headers := map[string]string{"PRIVATE-TOKEN": token}
	path := url.PathEscape(owner + "/" + name)
	var payload projectPayload
	err := provider.DoJSON(ctx, c.HTTP, http.MethodGet, c.BaseURL+"/api/v4/projects/"+path, headers, &payload)
	if err != nil {
		if errors.Is(err, provider.ErrNotFound) {
			return ports.RemoteStatus{Exists: false}, nil
		}
		return ports.RemoteStatus{}, err
	}
	return ports.RemoteStatus{
		Exists:   true,
		Private:  payload.Visibility != "public",
		CloneURL: payload.HTTPURLToRepo,
	}, nil
}

// CreateRepo creates a repository under the authenticated user.
func (c *Client) CreateRepo(ctx context.Context, token, owner, name string, private bool) (ports.RemoteStatus, error) {
	headers := map[string]string{"PRIVATE-TOKEN": token}
	visibility := "private"
	if !private {
		visibility = "public"
	}
	body := map[string]any{"name": name, "visibility": visibility}
	var payload projectPayload
	if err := provider.DoJSONBody(ctx, c.HTTP, http.MethodPost, c.BaseURL+"/api/v4/projects", headers, body, &payload); err != nil {
		return ports.RemoteStatus{}, err
	}
	cloneURL := payload.HTTPURLToRepo
	if cloneURL == "" && payload.PathWithNamespace != "" {
		cloneURL = strings.TrimSuffix(c.BaseURL, "/") + "/" + payload.PathWithNamespace + ".git"
	}
	return ports.RemoteStatus{Exists: true, Private: private, CloneURL: cloneURL}, nil
}
