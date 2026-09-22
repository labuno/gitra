package httptoken

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
)

type fakeSecrets struct{ values map[string]string }

func (f *fakeSecrets) Set(_ context.Context, ref, secret string) error {
	f.values[ref] = secret
	return nil
}
func (f *fakeSecrets) Get(_ context.Context, ref string) (string, error) {
	if value, ok := f.values[ref]; ok {
		return value, nil
	}
	return "", fmt.Errorf("%w: no stored credential for %s", domain.ErrAuthInvalid, ref)
}
func (f *fakeSecrets) Delete(_ context.Context, ref string) error {
	delete(f.values, ref)
	return nil
}

func httpsAccount() domain.Account {
	return domain.Account{
		ID:    "acc_https",
		Alias: "luna",
		Provider: domain.ProviderRef{
			Type:     domain.ProviderGitHub,
			Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategyHTTPSToken, Config: map[string]string{}},
		Revision:  1,
	}
}

func TestBuildGitConfigProjectsAccountScope(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{"github.com/lunafoundry": "gho_secret"}}
	strategy := New(secrets)
	entries, err := strategy.BuildGitConfig(auth.BuildRequest{Account: httpsAccount(), RemoteHost: "github.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Key != "credential.https://github.com.username" || entries[0].Value != "lunafoundry" {
		t.Fatalf("entry = %+v", entries[0])
	}
	if rendered := fmt.Sprint(entries); strings.Contains(rendered, "gho_secret") {
		t.Fatalf("token leaked into config entries: %s", rendered)
	}
}

func TestBuildGitConfigAddsHelperOnlyWhenRequested(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{"github.com/lunafoundry": "gho_secret"}}
	strategy := New(secrets)

	entries, err := strategy.BuildGitConfig(auth.BuildRequest{Account: httpsAccount(), RemoteHost: "github.com"})
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %+v, err = %v", entries, err)
	}
	entries, err = strategy.BuildGitConfig(auth.BuildRequest{
		Account: httpsAccount(), RemoteHost: "github.com", CredentialHelper: "osxkeychain",
	})
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries = %+v, err = %v", entries, err)
	}
	if entries[1].Key != "credential.helper" || entries[1].Value != "osxkeychain" {
		t.Fatalf("helper entry = %+v", entries[1])
	}
}

func TestValidateRequiresStoredSecret(t *testing.T) {
	empty := New(&fakeSecrets{values: map[string]string{}})
	if err := empty.Validate(context.Background(), httpsAccount()); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("missing secret error = %v, want ErrAuthInvalid", err)
	}
	withSecret := New(&fakeSecrets{values: map[string]string{"github.com/lunafoundry": "tok"}})
	if err := withSecret.Validate(context.Background(), httpsAccount()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	wrongStrategy := httpsAccount()
	wrongStrategy.Transport.Strategy = domain.StrategySSHKey
	if err := withSecret.Validate(context.Background(), wrongStrategy); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("wrong strategy error = %v, want ErrAuthInvalid", err)
	}
}

func TestTransportIsHTTPS(t *testing.T) {
	if got := New(&fakeSecrets{}).Transport(); got != domain.RemoteTransportHTTPS {
		t.Fatalf("Transport() = %q, want https", got)
	}
	_ = ports.GitConfigEntry{}
}
