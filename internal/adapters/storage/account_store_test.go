package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func testAccount(id domain.AccountID, alias string) domain.Account {
	return domain.Account{
		ID:    id,
		Alias: alias,
		Provider: domain.ProviderRef{
			Type:     domain.ProviderGitHub,
			Username: "lunafoundry",
			Endpoint: domain.ProviderEndpoint{Host: "github.com", SSHUser: "git", SSHPort: 22},
		},
		Identity:  domain.CommitIdentity{Name: "Luna", Email: "luna@example.com"},
		Transport: domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": "~/.ssh/id_ed25519_lunafoundry"}},
		Revision:  1,
	}
}

func TestAccountStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewAccountStore(dir)
	ctx := context.Background()
	account := testAccount("acc_001", "lunafoundry")

	if err := store.Save(ctx, account); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "accounts", "acc_001.json")); err != nil {
		t.Fatalf("account file must be named by ID: %v", err)
	}
	got, err := store.Get(ctx, "acc_001")
	if err != nil {
		t.Fatal(err)
	}
	if got.Alias != account.Alias || got.Provider.Endpoint.Host != "github.com" || got.Transport.Config["private_key"] == "" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	list, err := store.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List() = %v, %v", list, err)
	}
	if err := store.Delete(ctx, "acc_001"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "acc_001"); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Fatalf("Get after delete = %v, want ErrAccountNotFound", err)
	}
	if err := store.Delete(ctx, "acc_001"); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Fatalf("second delete = %v, want ErrAccountNotFound", err)
	}
}

func TestAccountStoreAliasUniqueness(t *testing.T) {
	dir := t.TempDir()
	store := NewAccountStore(dir)
	ctx := context.Background()
	if err := store.Save(ctx, testAccount("acc_001", "luna")); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, testAccount("acc_002", "luna")); !errors.Is(err, domain.ErrAccountExists) {
		t.Fatalf("duplicate alias error = %v, want ErrAccountExists", err)
	}
	// same ID may update itself
	updated := testAccount("acc_001", "luna")
	updated.Revision = 2
	if err := store.Save(ctx, updated); err != nil {
		t.Fatalf("update same id: %v", err)
	}
}

func TestAccountStoreRejectsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	store := NewAccountStore(dir)
	accountsDir := filepath.Join(dir, "accounts")
	if err := os.MkdirAll(accountsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(accountsDir, "acc_bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), "acc_bad"); err == nil {
		t.Fatal("Get() on corrupt file must fail")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("corrupt file must not be deleted: %v", err)
	}
}
