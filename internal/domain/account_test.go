package domain

import (
	"errors"
	"strings"
	"testing"
)

func validAccount() Account {
	return Account{
		ID:    "acc_01HXYZ",
		Alias: "lunafoundry",
		Provider: ProviderRef{
			Type:     ProviderGitHub,
			Username: "lunafoundry",
			Endpoint: ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: TransportConfig{Strategy: StrategySSHKey, Config: map[string]string{"private_key": "~/.ssh/id_ed25519_lunafoundry"}},
		Revision:  1,
	}
}

func TestAccountValidate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Account)
		wantErr   bool
		wantField string
	}{
		{"valid", func(a *Account) {}, false, ""},
		{"empty id", func(a *Account) { a.ID = "" }, true, "id"},
		{"bad id prefix", func(a *Account) { a.ID = "01HXYZ" }, true, "id"},
		{"empty alias", func(a *Account) { a.Alias = "" }, true, "alias"},
		{"uppercase alias", func(a *Account) { a.Alias = "Luna" }, true, "alias"},
		{"alias too long", func(a *Account) { a.Alias = strings.Repeat("a", 33) }, true, "alias"},
		{"alias with dash underscore ok", func(a *Account) { a.Alias = "luna_foundry-1" }, false, ""},
		{"port zero", func(a *Account) { a.Provider.Endpoint.SSHPort = 0 }, true, "provider.endpoint.ssh_port"},
		{"port too large", func(a *Account) { a.Provider.Endpoint.SSHPort = 65536 }, true, "provider.endpoint.ssh_port"},
		{"port 65535 ok", func(a *Account) { a.Provider.Endpoint.SSHPort = 65535 }, false, ""},
		{"unknown provider", func(a *Account) { a.Provider.Type = "bitbucket" }, true, "provider.type"},
		{"empty provider username", func(a *Account) { a.Provider.Username = "" }, true, "provider.username"},
		{"empty host", func(a *Account) { a.Provider.Endpoint.Host = "" }, true, "provider.endpoint.host"},
		{"empty git name", func(a *Account) { a.Identity.Name = "" }, true, "identity.name"},
		{"bad email", func(a *Account) { a.Identity.Email = "nope" }, true, "identity.email"},
		{"unknown transport strategy", func(a *Account) { a.Transport.Strategy = "oauth" }, true, "transport.strategy"},
		{"https-token transport ok", func(a *Account) {
			a.Transport.Strategy = StrategyHTTPSToken
			a.Transport.Config = map[string]string{}
		}, false, ""},
		{"missing private key", func(a *Account) { delete(a.Transport.Config, "private_key") }, true, "transport.config.private_key"},
		{"negative revision", func(a *Account) { a.Revision = -1 }, true, "revision"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := validAccount()
			tt.mutate(&account)
			err := account.Validate()
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() = nil, want error")
			}
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate() error %v does not wrap ErrInvalid", err)
			}
			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Validate() error %v is not *ValidationError", err)
			}
			if verr.Field != tt.wantField {
				t.Fatalf("ValidationError.Field = %q, want %q", verr.Field, tt.wantField)
			}
		})
	}
}
