package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

// sshaVerifyAccount is an ssh-key account with a real key file on disk.
func sshaVerifyAccount(t *testing.T) domain.Account {
	t.Helper()
	keyPath := filepath.Join(t.TempDir(), "id_ed25519_luna")
	if err := os.WriteFile(keyPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	return domain.Account{
		ID:    "acc_ssh_verify",
		Alias: "luna",
		Provider: domain.ProviderRef{
			Type:     domain.ProviderGitHub,
			Username: "luna",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": keyPath}},
		Revision:  1,
	}
}
