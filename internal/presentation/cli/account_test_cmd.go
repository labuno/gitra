package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zhanhd/gitra/internal/app"
	"github.com/zhanhd/gitra/internal/domain"
)

type verifyJSON struct {
	SchemaVersion    int    `json:"schema_version"`
	AccountID        string `json:"account_id"`
	Status           string `json:"status"`
	Success          bool   `json:"success"`
	ExpectedUsername string `json:"expected_username"`
	ActualUsername   string `json:"actual_username"`
	Message          string `json:"message"`
	Raw              string `json:"raw,omitempty"`
}

func (a *App) newAccountTestCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "test <alias>",
		Short: "Verify an account against its provider (explicit, online)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.App.Verification == nil {
				return fmt.Errorf("verification is not available in this build")
			}
			account, err := a.App.Accounts.GetByAlias(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			result, err := a.App.Verification.VerifyAccount(cmd.Context(), account.ID)
			if err != nil {
				return err
			}
			a.logf("verified %s: %s", account.Alias, result.Status)

			if jsonOut {
				payload := verifyJSON{
					SchemaVersion:    schemaVersion,
					AccountID:        string(account.ID),
					Status:           string(result.Status),
					Success:          result.Success,
					ExpectedUsername: result.ExpectedUsername,
					ActualUsername:   result.ActualUsername,
					Message:          result.Message,
				}
				if a.verbose {
					payload.Raw = result.Raw
				}
				if err := WriteJSON(a.Stdout, payload); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(a.Stdout, "%s (%s): %s\n",
					account.Alias, account.Provider.Endpoint.Host, result.Message); err != nil {
					return err
				}
				if a.verbose && result.Raw != "" {
					fmt.Fprintf(a.Stdout, "raw: %s\n", result.Raw)
				}
			}

			if !result.Success {
				return verifyError(result)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

// verifyError maps a failed verification to an exit-code-bearing error:
// authentication problems are exit 5, environment problems exit 1.
func verifyError(result app.VerificationResult) error {
	if result.Status == app.VerifyUnavailable {
		return fmt.Errorf("verification could not run: %s", result.Message)
	}
	return fmt.Errorf("%w: %s", domain.ErrAuthInvalid, result.Message)
}
