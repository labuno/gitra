package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/zhanhd/gitra/internal/domain"
)

// JSON DTOs follow the frozen contract (addendum appendix A.1):
// schema_version is top-level, fields are only added, never changed.
type providerJSON struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Username string `json:"username"`
}

type authJSON struct {
	Strategy string `json:"strategy"`
	State    string `json:"state"`
}

type accountItemJSON struct {
	ID           string       `json:"id"`
	Alias        string       `json:"alias"`
	Provider     providerJSON `json:"provider"`
	Auth         authJSON     `json:"auth"`
	ProjectCount int          `json:"project_count"`
}

type accountListJSON struct {
	SchemaVersion int               `json:"schema_version"`
	Accounts      []accountItemJSON `json:"accounts"`
}

type projectJSON struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type accountShowJSON struct {
	SchemaVersion int             `json:"schema_version"`
	Account       accountItemJSON `json:"account"`
	Projects      []projectJSON   `json:"projects"`
}

type accountMutationJSON struct {
	SchemaVersion int             `json:"schema_version"`
	Account       accountItemJSON `json:"account"`
}

// authState is a local, network-free verdict (baseline §4.8).
func (a *App) authState(ctx context.Context, account domain.Account) string {
	strategy, err := a.App.Deps.Auth.Get(account.Transport.Strategy)
	if err != nil {
		return "invalid"
	}
	if err := strategy.Validate(ctx, account); err != nil {
		return "invalid"
	}
	return "configured"
}

func (a *App) accountItem(ctx context.Context, account domain.Account, projectCount int) accountItemJSON {
	return accountItemJSON{
		ID:    string(account.ID),
		Alias: account.Alias,
		Provider: providerJSON{
			Type:     string(account.Provider.Type),
			Host:     account.Provider.Endpoint.Host,
			Username: account.Provider.Username,
		},
		Auth:         authJSON{Strategy: account.Transport.Strategy, State: a.authState(ctx, account)},
		ProjectCount: projectCount,
	}
}

func (a *App) writeAccountList(w io.Writer, items []accountItemJSON) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ALIAS\tID\tPROVIDER\tHOST\tPROJECTS"); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n",
			item.Alias, item.ID, item.Provider.Type, item.Provider.Host, item.ProjectCount); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func (a *App) writeAccountShow(w io.Writer, item accountItemJSON, projects []projectJSON) error {
	lines := []struct{ label, value string }{
		{"Alias", item.Alias},
		{"ID", item.ID},
		{"Provider", item.Provider.Type},
		{"Host", item.Provider.Host},
		{"Username", item.Provider.Username},
		{"Auth", item.Auth.Strategy + " (" + item.Auth.State + ")"},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "%-10s %s\n", line.label, line.value); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "Projects"); err != nil {
		return err
	}
	for _, project := range projects {
		if _, err := fmt.Fprintf(w, "  %s  %s\n", project.ID, project.Path); err != nil {
			return err
		}
	}
	return nil
}
