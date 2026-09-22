package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
	"github.com/zhanhd/gitra/internal/strategies/auth/httptoken"
	"github.com/zhanhd/gitra/internal/strategies/auth/sshkey"
)

type stubProfiles struct {
	profile ports.ProviderProfile
	err     error
}

func (s stubProfiles) Profile(context.Context, domain.ProviderType, string, string) (ports.ProviderProfile, error) {
	return s.profile, s.err
}

type stubSSH struct {
	result ports.SSHTestResult
	err    error
}

func (s stubSSH) Test(context.Context, ports.SSHTestRequest) (ports.SSHTestResult, error) {
	return s.result, s.err
}

type stubParser struct {
	username string
	ok       bool
}

func (s stubParser) ParseSSHIdentity(context.Context, domain.ProviderType, string, string) (string, bool) {
	return s.username, s.ok
}

func verifyEnv(t *testing.T, account domain.Account, secrets *fakeSecrets, profiles stubProfiles, ssh stubSSH, parser stubParser) *VerificationService {
	t.Helper()
	registry := auth.NewRegistry()
	if err := registry.Register(sshkey.New()); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(httptoken.New(secrets)); err != nil {
		t.Fatal(err)
	}
	deps := Deps{
		Accounts: &fakeAccounts{account: account},
		Auth:     registry,
		Secrets:  secrets,
		SSH:      ssh,
	}
	return NewVerificationService(deps, profiles, parser)
}

func httpsVerifyAccount() domain.Account {
	account := httpsAccount()
	account.ID = "acc_verify"
	return account
}

func TestVerifyHTTPSOutcomes(t *testing.T) {
	ctx := context.Background()

	ok := verifyEnv(t, httpsVerifyAccount(),
		&fakeSecrets{values: map[string]string{"github.com/lunafoundry": "tok"}},
		stubProfiles{profile: ports.ProviderProfile{Username: "lunafoundry"}}, stubSSH{}, stubParser{})
	result, err := ok.VerifyAccount(ctx, "acc_verify")
	if err != nil || !result.Success || result.Status != VerifyOK {
		t.Fatalf("ok result = %+v, %v", result, err)
	}

	mismatch := verifyEnv(t, httpsVerifyAccount(),
		&fakeSecrets{values: map[string]string{"github.com/lunafoundry": "tok"}},
		stubProfiles{profile: ports.ProviderProfile{Username: "someone-else"}}, stubSSH{}, stubParser{})
	result, err = mismatch.VerifyAccount(ctx, "acc_verify")
	if err != nil || result.Status != VerifyMismatch || result.ActualUsername != "someone-else" {
		t.Fatalf("mismatch result = %+v, %v", result, err)
	}

	authFailed := verifyEnv(t, httpsVerifyAccount(),
		&fakeSecrets{values: map[string]string{"github.com/lunafoundry": "tok"}},
		stubProfiles{err: fmt.Errorf("%w: rejected", domain.ErrAuthInvalid)}, stubSSH{}, stubParser{})
	result, _ = authFailed.VerifyAccount(ctx, "acc_verify")
	if result.Status != VerifyAuthFailed {
		t.Fatalf("auth failed result = %+v", result)
	}

	unavailable := verifyEnv(t, httpsVerifyAccount(),
		&fakeSecrets{values: map[string]string{"github.com/lunafoundry": "tok"}},
		stubProfiles{err: errors.New("provider returned HTTP 500")}, stubSSH{}, stubParser{})
	result, _ = unavailable.VerifyAccount(ctx, "acc_verify")
	if result.Status != VerifyUnavailable {
		t.Fatalf("unavailable result = %+v", result)
	}

	noCredential := verifyEnv(t, httpsVerifyAccount(), &fakeSecrets{values: map[string]string{}}, stubProfiles{}, stubSSH{}, stubParser{})
	result, _ = noCredential.VerifyAccount(ctx, "acc_verify")
	if result.Status != VerifyLocalInvalid {
		t.Fatalf("missing credential result = %+v", result)
	}
}

func TestVerifySSHOutcomes(t *testing.T) {
	ctx := context.Background()
	account := sshaVerifyAccount(t)

	ok := verifyEnv(t, account, &fakeSecrets{}, stubProfiles{},
		stubSSH{result: ports.SSHTestResult{ExitCode: 1, Stderr: "Hi luna! You've successfully authenticated"}},
		stubParser{username: "luna", ok: true})
	result, err := ok.VerifyAccount(ctx, account.ID)
	if err != nil || !result.Success || result.Status != VerifyOK {
		t.Fatalf("ssh ok result = %+v, %v", result, err)
	}
	if result.Raw == "" {
		t.Fatal("raw output must be preserved for --verbose")
	}

	unrecognized := verifyEnv(t, account, &fakeSecrets{}, stubProfiles{},
		stubSSH{result: ports.SSHTestResult{ExitCode: 255, Stderr: "weird banner"}}, stubParser{ok: false})
	result, _ = unrecognized.VerifyAccount(ctx, account.ID)
	if result.Status != VerifyUnrecognized {
		t.Fatalf("unrecognized result = %+v", result)
	}

	mismatch := verifyEnv(t, account, &fakeSecrets{}, stubProfiles{},
		stubSSH{result: ports.SSHTestResult{ExitCode: 1}}, stubParser{username: "other", ok: true})
	result, _ = mismatch.VerifyAccount(ctx, account.ID)
	if result.Status != VerifyMismatch {
		t.Fatalf("mismatch result = %+v", result)
	}

	unavailable := verifyEnv(t, account, &fakeSecrets{}, stubProfiles{},
		stubSSH{err: errors.New("ssh binary missing")}, stubParser{})
	result, _ = unavailable.VerifyAccount(ctx, account.ID)
	if result.Status != VerifyUnavailable {
		t.Fatalf("unavailable result = %+v", result)
	}

	// Local configuration invalid: the key file no longer exists.
	missingKey := account
	missingKey.Transport.Config = map[string]string{"private_key": "/nonexistent/key"}
	invalid := verifyEnv(t, missingKey, &fakeSecrets{}, stubProfiles{}, stubSSH{}, stubParser{})
	result, _ = invalid.VerifyAccount(ctx, missingKey.ID)
	if result.Status != VerifyLocalInvalid {
		t.Fatalf("local invalid result = %+v", result)
	}
}
