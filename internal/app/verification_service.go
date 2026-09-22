package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// VerificationStatus classifies verification outcomes (baseline §48).
type VerificationStatus string

const (
	VerifyOK           VerificationStatus = "ok"
	VerifyLocalInvalid VerificationStatus = "local_config_invalid"
	VerifyAuthFailed   VerificationStatus = "authentication_failed"
	VerifyMismatch     VerificationStatus = "username_mismatch"
	VerifyUnavailable  VerificationStatus = "provider_unavailable"
	VerifyUnrecognized VerificationStatus = "unrecognized_response"
)

// VerificationResult is the outcome of one explicit account test. A non-ok
// status is a normal result, not a Go error: the local configuration may be
// fine while the credential was revoked.
type VerificationResult struct {
	Status           VerificationStatus
	Success          bool
	ExpectedUsername string
	ActualUsername   string
	Message          string
	Raw              string
}

// VerificationService performs explicit, user-triggered account tests only
// (baseline §4.8: never on TUI startup).
type VerificationService struct {
	deps     Deps
	profiles ports.ProfileProvider
	parsers  ports.SSHIdentityParser
}

// NewVerificationService wires the verification use case.
func NewVerificationService(deps Deps, profiles ports.ProfileProvider, parsers ports.SSHIdentityParser) *VerificationService {
	return &VerificationService{deps: deps, profiles: profiles, parsers: parsers}
}

// VerifyAccount verifies one account using its auth strategy.
func (s *VerificationService) VerifyAccount(ctx context.Context, id domain.AccountID) (VerificationResult, error) {
	account, err := s.deps.Accounts.Get(ctx, id)
	if err != nil {
		return VerificationResult{}, err
	}
	strategy, err := s.deps.Auth.Get(account.Transport.Strategy)
	if err != nil {
		return VerificationResult{}, err
	}
	expected := account.Provider.Username
	base := VerificationResult{ExpectedUsername: expected, ActualUsername: expected}

	if err := strategy.Validate(ctx, account); err != nil {
		base.Status = VerifyLocalInvalid
		base.Message = localInvalidMessage(account, err)
		return base, nil
	}

	switch strategy.Transport() {
	case domain.RemoteTransportHTTPS:
		return s.verifyHTTPS(ctx, account, base)
	case domain.RemoteTransportSSH:
		return s.verifySSH(ctx, account, base)
	default:
		base.Status = VerifyUnrecognized
		base.Message = "unsupported transport"
		return base, nil
	}
}

func (s *VerificationService) verifyHTTPS(ctx context.Context, account domain.Account, base VerificationResult) (VerificationResult, error) {
	if s.profiles == nil || s.deps.Secrets == nil {
		base.Status = VerifyUnavailable
		base.Message = "provider API is not configured in this build"
		return base, nil
	}
	secret, err := s.deps.Secrets.Get(ctx, account.CredentialRef())
	if err != nil {
		base.Status = VerifyLocalInvalid
		base.Message = fmt.Sprintf("no stored credential; run 'gitra login --provider %s'", account.Provider.Type)
		return base, nil
	}
	profile, err := s.profiles.Profile(ctx, account.Provider.Type, account.Provider.Endpoint.Host, secret)
	if err != nil {
		if errors.Is(err, domain.ErrAuthInvalid) {
			base.Status = VerifyAuthFailed
			base.Message = "the provider rejected the stored credential; run 'gitra login' again"
			return base, nil
		}
		base.Status = VerifyUnavailable
		base.Message = err.Error()
		return base, nil
	}
	base.ActualUsername = profile.Username
	if !strings.EqualFold(profile.Username, base.ExpectedUsername) {
		base.Status = VerifyMismatch
		base.Message = fmt.Sprintf("authenticated as %s, but the account expects %s", profile.Username, base.ExpectedUsername)
		return base, nil
	}
	base.Status = VerifyOK
	base.Success = true
	base.Message = "authenticated"
	return base, nil
}

func (s *VerificationService) verifySSH(ctx context.Context, account domain.Account, base VerificationResult) (VerificationResult, error) {
	if s.deps.SSH == nil || s.parsers == nil {
		base.Status = VerifyUnavailable
		base.Message = "ssh client is not configured in this build"
		return base, nil
	}
	result, err := s.deps.SSH.Test(ctx, ports.SSHTestRequest{
		Host:           account.Provider.Endpoint.Host,
		User:           account.Provider.Endpoint.SSHUser,
		Port:           account.Provider.Endpoint.SSHPort,
		PrivateKeyPath: account.Transport.Config["private_key"],
	})
	if err != nil {
		base.Status = VerifyUnavailable
		base.Message = fmt.Sprintf("could not run ssh: %v", err)
		return base, nil
	}
	base.Raw = strings.TrimSpace(result.Stdout + "\n" + result.Stderr)

	username, ok := s.parsers.ParseSSHIdentity(ctx, account.Provider.Type, result.Stdout, result.Stderr)
	if !ok {
		base.Status = VerifyUnrecognized
		base.Message = "could not recognize the provider response (rerun with --verbose to inspect it)"
		return base, nil
	}
	base.ActualUsername = username
	if !strings.EqualFold(username, base.ExpectedUsername) {
		base.Status = VerifyMismatch
		base.Message = fmt.Sprintf("authenticated as %s, but the account expects %s", username, base.ExpectedUsername)
		return base, nil
	}
	base.Status = VerifyOK
	base.Success = true
	base.Message = "authenticated"
	return base, nil
}

func localInvalidMessage(account domain.Account, err error) string {
	if account.Transport.Strategy == domain.StrategyHTTPSToken {
		return fmt.Sprintf("no stored credential; run 'gitra login --provider %s'", account.Provider.Type)
	}
	return err.Error()
}
