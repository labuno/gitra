package sshkey

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/strategies/auth"
)

func testAccount(t *testing.T, keyPath string, port int) domain.Account {
	t.Helper()
	return domain.Account{
		ID:    "acc_001",
		Alias: "luna",
		Provider: domain.ProviderRef{
			Type:     domain.ProviderGitHub,
			Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "git.example.com", SSHUser: "git", SSHPort: port},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": keyPath}},
		Revision:  1,
	}
}

func newKey(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "id_ed25519_luna")
	if err := os.WriteFile(path, []byte("PRIVATE KEY PLACEHOLDER"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildGitConfigPortRules(t *testing.T) {
	key := newKey(t)
	strategy := New()
	tests := []struct {
		name             string
		port             int
		remoteExplicit   bool
		wantContainsPort bool
	}{
		{"default port", 22, false, false},
		{"custom port scp-like", 2222, false, true},
		{"custom port but url carries it", 2222, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := testAccount(t, key, tt.port)
			entries, err := strategy.BuildGitConfig(auth.BuildRequest{Account: account, RemoteHasExplicitPort: tt.remoteExplicit})
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Key != "core.sshCommand" {
				t.Fatalf("entries = %+v", entries)
			}
			value := entries[0].Value
			if !strings.Contains(value, "IdentitiesOnly=yes") || !strings.Contains(value, key) {
				t.Fatalf("sshCommand = %q", value)
			}
			if strings.Contains(value, "-p ") != tt.wantContainsPort {
				t.Fatalf("sshCommand = %q, wantContainsPort=%v", value, tt.wantContainsPort)
			}
		})
	}
}

func TestBuildGitConfigQuotesPathsWithSpaces(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "my keys", "id_ed25519")
	if err := os.MkdirAll(filepath.Dir(key), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := New().BuildGitConfig(auth.BuildRequest{Account: testAccount(t, key, 22)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(entries[0].Value, `"`+key+`"`) {
		t.Fatalf("path with spaces must be quoted: %q", entries[0].Value)
	}
}

func TestValidateRejectsMissingKey(t *testing.T) {
	account := testAccount(t, filepath.Join(t.TempDir(), "missing"), 22)
	if err := New().Validate(context.Background(), account); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("error = %v, want ErrAuthInvalid", err)
	}
	if _, err := New().BuildGitConfig(auth.BuildRequest{Account: account}); !errors.Is(err, domain.ErrAuthInvalid) {
		t.Fatalf("build error = %v, want ErrAuthInvalid", err)
	}
}
