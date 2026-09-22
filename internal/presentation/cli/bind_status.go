package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/domain"
)

type bindingJSON struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Strategy  string `json:"strategy"`
	Health    string `json:"health"`
}

type statusAccountJSON struct {
	ID       string       `json:"id"`
	Alias    string       `json:"alias"`
	Provider providerJSON `json:"provider"`
}

type statusJSON struct {
	SchemaVersion int                `json:"schema_version"`
	Repository    string             `json:"repository"`
	Bound         bool               `json:"bound"`
	Binding       *bindingJSON       `json:"binding,omitempty"`
	Account       *statusAccountJSON `json:"account,omitempty"`
	Auth          *authJSON          `json:"auth,omitempty"`
}

type bindJSON struct {
	SchemaVersion int         `json:"schema_version"`
	Binding       bindingJSON `json:"binding"`
	Repository    string      `json:"repository"`
}

// pathArg resolves an optional path argument, defaulting to the cwd.
func pathArg(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	return os.Getwd()
}

func (a *App) newBindCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "bind <alias> [path]",
		Short: "Bind a repository to an account",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			account, err := a.App.Accounts.GetByAlias(ctx, args[0])
			if err != nil {
				return err
			}
			path, err := pathArg(args[1:])
			if err != nil {
				return err
			}
			binding, err := a.App.Bindings.Bind(ctx, app.BindRequest{AccountID: account.ID, Path: path})
			if err != nil {
				return err
			}
			a.logf("bound %s to %s", binding.Repository.Path, account.Alias)
			if jsonOut {
				return WriteJSON(a.Stdout, bindJSON{
					SchemaVersion: schemaVersion,
					Repository:    binding.Repository.Path,
					Binding: bindingJSON{
						ID: string(binding.ID), AccountID: string(binding.AccountID),
						Strategy: binding.Strategy, Health: string(domain.HealthOK),
					},
				})
			}
			_, err = fmt.Fprintf(a.Stdout, "bound %s to %s (%s)\n", binding.Repository.Path, account.Alias, binding.ID)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func (a *App) newUnbindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbind [path]",
		Short: "Restore a repository to its pre-bind state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := pathArg(args)
			if err != nil {
				return err
			}
			if err := a.App.Bindings.Unbind(cmd.Context(), app.UnbindRequest{Path: path}); err != nil {
				return err
			}
			a.logf("unbound %s", path)
			_, err = fmt.Fprintf(a.Stdout, "unbound %s\n", path)
			return err
		},
	}
	return cmd
}

func (a *App) newStatusCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status [path]",
		Short: "Show binding status and health for a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			path, err := pathArg(args)
			if err != nil {
				return err
			}
			status, err := a.App.Bindings.Status(ctx, path)
			if err != nil {
				return err
			}

			if jsonOut {
				payload := statusJSON{SchemaVersion: schemaVersion, Repository: status.Repository, Bound: status.Bound}
				if status.Bound {
					payload.Binding = &bindingJSON{
						ID: string(status.Binding.ID), AccountID: string(status.Binding.AccountID),
						Strategy: status.Binding.Strategy, Health: string(status.Health),
					}
					payload.Account = &statusAccountJSON{
						ID:    string(status.Account.ID),
						Alias: status.Account.Alias,
						Provider: providerJSON{
							Type:     string(status.Account.Provider.Type),
							Host:     status.Account.Provider.Endpoint.Host,
							Username: status.Account.Provider.Username,
						},
					}
					state := "invalid"
					switch {
					case status.NeedsLogin:
						state = "needs_login"
					case status.Health == domain.HealthOK || status.Health == domain.HealthDrift:
						state = a.authState(ctx, status.Account)
					}
					payload.Auth = &authJSON{Strategy: status.Account.Transport.Strategy, State: state}
				}
				if err := WriteJSON(a.Stdout, payload); err != nil {
					return err
				}
			} else if !status.Bound {
				if _, err := fmt.Fprintf(a.Stdout, "not bound: %s\n", status.Repository); err != nil {
					return err
				}
			} else {
				lines := []struct{ label, value string }{
					{"repository", status.Repository},
					{"binding", string(status.Binding.ID)},
					{"account", fmt.Sprintf("%s (%s)", status.Account.Alias, status.Account.ID)},
					{"health", string(status.Health)},
				}
				for _, line := range lines {
					if _, err := fmt.Fprintf(a.Stdout, "%-10s %s\n", line.label, line.value); err != nil {
						return err
					}
				}
			}

			if status.Bound && status.Health != domain.HealthOK {
				return fmt.Errorf("%w: repository %s reports %s (run 'gitra status' after fixing, or re-bind with 'gitra bind')",
					domain.ErrStateDrift, status.Repository, status.Health)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

var _ = errors.Is
