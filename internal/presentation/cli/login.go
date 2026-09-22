package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/domain"
)

type loginJSON struct {
	SchemaVersion int             `json:"schema_version"`
	Account       accountItemJSON `json:"account"`
	Credential    struct {
		Host   string `json:"host"`
		Source string `json:"source"`
	} `json:"credential"`
}

func (a *App) newLoginCmd() *cobra.Command {
	var (
		providerType, host, alias string
		tokenStdin, jsonOut       bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to a Git provider and register the account",
		Long: "Log in without creating SSH keys:\n" +
			"  * reuse an existing gh/glab session, or\n" +
			"  * paste a personal access token via --token-stdin (or GITRA_TOKEN).\n" +
			"The credential is stored in your system credential store; repositories only\n" +
			"record which account to use.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a.App.Login == nil {
				return fmt.Errorf("login is not available in this build")
			}
			provider, err := parseProviderType(providerType)
			if err != nil {
				return err
			}
			result, err := a.App.Login.Login(cmd.Context(), app.LoginRequest{
				Provider:      provider,
				Host:          host,
				Alias:         alias,
				AllowStdin:    tokenStdin,
				AllowCLIReuse: true,
			})
			if err != nil {
				// The account may still have been saved when reconciliation
				// reported problems; surface that explicitly.
				if result.Account.ID != "" {
					fmt.Fprintf(a.Stderr, "account %s was saved, but some repositories need attention\n", result.Account.Alias)
				}
				return err
			}
			a.logf("logged in as %s via %s", result.Profile.Username, result.TokenSource)

			item := a.accountItem(cmd.Context(), result.Account, 0)
			if jsonOut {
				payload := loginJSON{SchemaVersion: schemaVersion, Account: item}
				payload.Credential.Host = result.Account.Provider.Endpoint.Host
				payload.Credential.Source = result.TokenSource
				return WriteJSON(a.Stdout, payload)
			}
			action := "created"
			if !result.Created {
				action = "updated"
			}
			if _, err := fmt.Fprintf(a.Stdout, "logged in as %s (%s) — %s account %s (%s)\n",
				result.Profile.Username, result.Account.Provider.Endpoint.Host, action, result.Account.Alias, result.Account.ID); err != nil {
				return err
			}
			if result.Profile.EmailFallback {
				fmt.Fprintf(a.Stderr,
					"note: the provider exposed no verified email; using %s (change it with 'gitra account edit %s --email ...')\n",
					result.Account.Identity.Email, result.Account.Alias)
			}
			fmt.Fprintf(a.Stdout, "next: gitra bind %s [path]\n", result.Account.Alias)
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&providerType, "provider", "", "provider type: github|gitlab|gitea")
	flags.StringVar(&host, "host", "", "provider host (defaults to the provider default)")
	flags.StringVar(&alias, "alias", "", "local alias (defaults to the provider username)")
	flags.BoolVar(&tokenStdin, "token-stdin", false, "read a personal access token from stdin")
	flags.BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func (a *App) newLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout <alias>",
		Short: "Delete the stored provider credential of an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.App.Login == nil {
				return fmt.Errorf("logout is not available in this build")
			}
			account, err := a.App.Accounts.GetByAlias(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := a.App.Login.Logout(cmd.Context(), account.ID); err != nil {
				return err
			}
			a.logf("logged out %s", account.Alias)
			if _, err := fmt.Fprintf(a.Stdout, "logged out %s (%s)\n", account.Alias, account.Provider.Endpoint.Host); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.Stdout, "remove the token in the provider if you no longer need it: %s\n", revokeHint(account.Provider.Type))
			return err
		},
	}
	return cmd
}

func revokeHint(providerType domain.ProviderType) string {
	switch providerType {
	case domain.ProviderGitHub:
		return "https://github.com/settings/tokens"
	case domain.ProviderGitLab:
		return "https://gitlab.com/-/user_settings/personal_access_tokens"
	default:
		return "your provider's token settings"
	}
}
