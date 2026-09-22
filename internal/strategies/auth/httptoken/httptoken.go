// Package httptoken implements the V1.1 https-token auth strategy: the
// repository only learns which account to use; the secret itself stays in the
// system credential store and is fetched by git's own helper.
package httptoken

import (
	"context"
	"fmt"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
)

// ID is the strategy identifier stored on the account.
const ID = domain.StrategyHTTPSToken

// Strategy projects credential scope entries for HTTPS remotes.
type Strategy struct {
	secrets ports.SecretStore
}

// New builds the strategy over a secret store.
func New(secrets ports.SecretStore) *Strategy { return &Strategy{secrets: secrets} }

// ID implements auth.Strategy.
func (s *Strategy) ID() string { return ID }

// Transport implements auth.Strategy.
func (s *Strategy) Transport() domain.RemoteTransport { return domain.RemoteTransportHTTPS }

// CredentialRef is the stable secret reference: "<host>/<username>".
func CredentialRef(account domain.Account) string { return account.CredentialRef() }

// Validate checks the account strategy and that a credential is stored.
func (s *Strategy) Validate(ctx context.Context, account domain.Account) error {
	if account.Transport.Strategy != ID {
		return fmt.Errorf("%w: account %s uses transport %q", domain.ErrAuthInvalid, account.ID, account.Transport.Strategy)
	}
	if s.secrets == nil {
		return fmt.Errorf("%w: no secret store configured", domain.ErrAuthInvalid)
	}
	if _, err := s.secrets.Get(ctx, CredentialRef(account)); err != nil {
		return fmt.Errorf("%w: no stored credential for %s", domain.ErrAuthInvalid, CredentialRef(account))
	}
	return nil
}

// BuildGitConfig returns the managed credential scope entries. The token value
// never appears here.
func (s *Strategy) BuildGitConfig(req auth.BuildRequest) ([]ports.GitConfigEntry, error) {
	if err := s.Validate(context.Background(), req.Account); err != nil {
		return nil, err
	}
	url := "https://" + req.Account.Provider.Endpoint.Host
	entries := []ports.GitConfigEntry{
		{Key: "credential." + url + ".username", Value: req.Account.Provider.Username},
	}
	if req.CredentialHelper != "" {
		entries = append(entries, ports.GitConfigEntry{Key: "credential.helper", Value: req.CredentialHelper})
	}
	return entries, nil
}
