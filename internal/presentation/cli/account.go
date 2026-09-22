package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/domain"
)

func (a *App) newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage Git accounts",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		a.newAccountAddCmd(),
		a.newAccountListCmd(),
		a.newAccountShowCmd(),
		a.newAccountEditCmd(),
		a.newAccountRemoveCmd(),
		a.newAccountTestCmd(),
	)
	return cmd
}

func parseProviderType(value string) (domain.ProviderType, error) {
	provider := domain.ProviderType(value)
	if !provider.Valid() {
		return "", fmt.Errorf("%w: unknown provider %q (github|gitlab|gitea)", errUsage, value)
	}
	return provider, nil
}

func (a *App) newAccountAddCmd() *cobra.Command {
	var (
		alias, providerType, host, sshUser, username, name, email, auth, key string
		sshPort                                                              int
		jsonOut                                                              bool
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an account (non-interactive)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider, err := parseProviderType(providerType)
			if err != nil {
				return err
			}
			for flag, value := range map[string]string{
				"alias": alias, "username": username, "name": name, "email": email, "key": key,
			} {
				if value == "" {
					return fmt.Errorf("%w: --%s is required", errUsage, flag)
				}
			}
			if auth == "" {
				auth = domain.StrategySSHKey
			}
			if auth != domain.StrategySSHKey {
				return fmt.Errorf("%w: --auth %q is not supported in V1 (use ssh-key)", errUsage, auth)
			}
			endpoint, ok := domain.DefaultEndpoint(provider)
			if !ok {
				return fmt.Errorf("%w: unknown provider %q", errUsage, providerType)
			}
			if host != "" {
				endpoint.Host = host
			}
			if sshUser != "" {
				endpoint.SSHUser = sshUser
			}
			if sshPort != 0 {
				endpoint.SSHPort = sshPort
			}

			account, err := a.App.Accounts.Create(cmd.Context(), app.CreateAccountRequest{
				Alias: alias,
				Provider: domain.ProviderRef{
					Type:     provider,
					Username: username,
					Endpoint: endpoint,
				},
				Identity:  domain.CommitIdentity{Name: name, Email: email},
				Transport: domain.TransportConfig{Strategy: auth, Config: map[string]string{"private_key": key}},
			})
			if err != nil {
				return err
			}
			a.logf("created account %s", account.Alias)
			if jsonOut {
				return WriteJSON(a.Stdout, accountMutationJSON{SchemaVersion: schemaVersion, Account: a.accountItem(cmd.Context(), account, 0)})
			}
			_, err = fmt.Fprintf(a.Stdout, "created account %s (%s)\n", account.Alias, account.ID)
			return err
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&alias, "alias", "", "local alias used by this machine")
	flags.StringVar(&providerType, "provider", "", "provider type: github|gitlab|gitea")
	flags.StringVar(&host, "host", "", "SSH host (defaults to the provider default)")
	flags.StringVar(&sshUser, "ssh-user", "", "SSH user (defaults to the provider default)")
	flags.IntVar(&sshPort, "ssh-port", 0, "SSH port (defaults to the provider default)")
	flags.StringVar(&username, "username", "", "provider account username")
	flags.StringVar(&name, "name", "", "git commit name")
	flags.StringVar(&email, "email", "", "git commit email")
	flags.StringVar(&auth, "auth", domain.StrategySSHKey, "transport auth strategy (ssh-key)")
	flags.StringVar(&key, "key", "", "path to the SSH private key")
	flags.BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

// accountProjects lists the bindings of one account.
func (a *App) accountProjects(ctx context.Context, id domain.AccountID) ([]domain.RepositoryBinding, error) {
	return a.App.Deps.Bindings.ListByAccount(ctx, id)
}

func (a *App) newAccountListCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			accounts, err := a.App.Accounts.List(ctx)
			if err != nil {
				return err
			}
			items := make([]accountItemJSON, 0, len(accounts))
			for _, account := range accounts {
				projects, err := a.accountProjects(ctx, account.ID)
				if err != nil {
					return err
				}
				items = append(items, a.accountItem(ctx, account, len(projects)))
			}
			if jsonOut {
				return WriteJSON(a.Stdout, accountListJSON{SchemaVersion: schemaVersion, Accounts: items})
			}
			return a.writeAccountList(a.Stdout, items)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func (a *App) newAccountShowCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "show <alias>",
		Short: "Show one account and its bound repositories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			account, err := a.App.Accounts.GetByAlias(ctx, args[0])
			if err != nil {
				return err
			}
			bindings, err := a.accountProjects(ctx, account.ID)
			if err != nil {
				return err
			}
			projects := make([]projectJSON, 0, len(bindings))
			for _, binding := range bindings {
				projects = append(projects, projectJSON{ID: string(binding.ID), Path: binding.Repository.Path})
			}
			item := a.accountItem(ctx, account, len(projects))
			if jsonOut {
				return WriteJSON(a.Stdout, accountShowJSON{SchemaVersion: schemaVersion, Account: item, Projects: projects})
			}
			return a.writeAccountShow(a.Stdout, item, projects)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func (a *App) newAccountEditCmd() *cobra.Command {
	var (
		alias, name, email, username, host, sshUser, key string
		sshPort                                          int
		jsonOut                                          bool
	)
	cmd := &cobra.Command{
		Use:   "edit <alias>",
		Short: "Edit an account and reconcile every bound repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			account, err := a.App.Accounts.GetByAlias(ctx, args[0])
			if err != nil {
				return err
			}
			changed := false
			if cmd.Flags().Changed("alias") {
				account.Alias = alias
				changed = true
			}
			if cmd.Flags().Changed("name") {
				account.Identity.Name = name
				changed = true
			}
			if cmd.Flags().Changed("email") {
				account.Identity.Email = email
				changed = true
			}
			if cmd.Flags().Changed("username") {
				account.Provider.Username = username
				changed = true
			}
			if cmd.Flags().Changed("host") {
				account.Provider.Endpoint.Host = host
				changed = true
			}
			if cmd.Flags().Changed("ssh-user") {
				account.Provider.Endpoint.SSHUser = sshUser
				changed = true
			}
			if cmd.Flags().Changed("ssh-port") {
				account.Provider.Endpoint.SSHPort = sshPort
				changed = true
			}
			if cmd.Flags().Changed("key") {
				account.Transport.Config["private_key"] = key
				changed = true
			}
			if !changed {
				return fmt.Errorf("%w: no changes requested (see gitra account edit --help)", errUsage)
			}

			updated, err := a.App.Accounts.Update(ctx, app.UpdateAccountRequest{Account: account})
			if err != nil {
				if updated.ID != "" {
					fmt.Fprintf(a.Stderr, "account %s was saved, but some repositories need attention\n", updated.Alias)
				}
				return err
			}
			a.logf("updated account %s", updated.Alias)
			if jsonOut {
				return WriteJSON(a.Stdout, accountMutationJSON{SchemaVersion: schemaVersion, Account: a.accountItem(ctx, updated, 0)})
			}
			_, err = fmt.Fprintf(a.Stdout, "updated account %s (%s)\n", updated.Alias, updated.ID)
			return err
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&alias, "alias", "", "new alias")
	flags.StringVar(&name, "name", "", "git commit name")
	flags.StringVar(&email, "email", "", "git commit email")
	flags.StringVar(&username, "username", "", "provider account username")
	flags.StringVar(&host, "host", "", "SSH host")
	flags.StringVar(&sshUser, "ssh-user", "", "SSH user")
	flags.IntVar(&sshPort, "ssh-port", 0, "SSH port")
	flags.StringVar(&key, "key", "", "path to the SSH private key")
	flags.BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func (a *App) newAccountRemoveCmd() *cobra.Command {
	var unbindAll bool
	cmd := &cobra.Command{
		Use:   "remove <alias>",
		Short: "Remove an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			account, err := a.App.Accounts.GetByAlias(ctx, args[0])
			if err != nil {
				return err
			}
			if err := a.App.Accounts.Delete(ctx, app.DeleteAccountRequest{ID: account.ID, UnbindAll: unbindAll}); err != nil {
				if errors.Is(err, domain.ErrAccountInUse) {
					return fmt.Errorf("%w; retry with --unbind-all to unbind them first", err)
				}
				return err
			}
			a.logf("removed account %s", account.Alias)
			_, err = fmt.Fprintf(a.Stdout, "removed account %s\n", account.Alias)
			return err
		},
	}
	cmd.Flags().BoolVar(&unbindAll, "unbind-all", false, "unbind every repository before removing")
	return cmd
}

var _ = os.Getenv // keep os import stable for future platform checks
