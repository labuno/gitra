package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/domain"
)

func TestBindManyHandlesMixedFolders(t *testing.T) {
	ctx := context.Background()
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")
	service := env.service

	// One bindable folder, the same folder again, and one path that is not a
	// repository. (The missing-remote case is covered below.)
	env.git.missing["/missing"] = true
	results, err := service.BindMany(ctx, []domain.AccountID{env.account.ID}, []string{"/repo", "/repo", "/missing"})
	if err != nil {
		t.Fatalf("BindMany() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Status != "bound" {
		t.Fatalf("first result = %+v, want bound", results[0])
	}
	byStatus := map[string]int{}
	for _, result := range results {
		byStatus[result.Status]++
	}
	// /repo binds once, the second attempt reports "already", /missing is skipped.
	if byStatus["bound"] != 1 || byStatus["already"] != 1 || byStatus["skipped"] != 1 {
		t.Fatalf("status counts = %v (results=%+v)", byStatus, results)
	}
	if len(env.bindings.saved) != 1 {
		t.Fatalf("bindings = %+v", env.bindings.saved)
	}
}

func TestBindManyReportsMissingRemote(t *testing.T) {
	ctx := context.Background()
	env := newHTTPSEnv(t, "") // repository without any remote
	env.git.remote = ""
	results, err := env.service.BindMany(ctx, []domain.AccountID{env.account.ID}, []string{"/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != "skipped" {
		t.Fatalf("results = %+v", results)
	}
	if !errors.Is(results[0].Err, domain.ErrUnsupportedRemote) {
		t.Fatalf("reason = %v, want ErrUnsupportedRemote", results[0].Err)
	}
}

// TestBindManyPicksTheAccountThatMatchesTheRemote covers the real-world mix:
// HTTPS repositories and SSH repositories bound in one batch with an SSH
// account selected first.
func TestBindManyPicksTheAccountThatMatchesTheRemote(t *testing.T) {
	ctx := context.Background()
	env := newHTTPSEnv(t, "https://github.com/lunafoundry/luna-site.git")

	// A second account that speaks SSH.
	sshAccount := env.account
	sshAccount.ID = "acc_ssh"
	sshAccount.Alias = "ssh-user"
	sshAccount.Transport = domain.TransportConfig{Strategy: domain.StrategySSHKey, Config: map[string]string{"private_key": "/tmp/key"}}

	accounts := &multiAccounts{primary: sshAccount, extra: []domain.Account{env.account}}
	env.deps.Accounts = accounts
	service := NewBindingService(env.deps)

	// The SSH account is the preference, but the repository remote is HTTPS.
	results, err := service.BindMany(ctx, []domain.AccountID{sshAccount.ID, env.account.ID}, []string{"/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != "bound" {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Account != env.account.Alias {
		t.Fatalf("account = %q, want the HTTPS account %q", results[0].Account, env.account.Alias)
	}
}

func TestBindManyExplainsTransportMismatch(t *testing.T) {
	ctx := context.Background()
	env := newHTTPSEnv(t, "git@github.com:lunafoundry/luna-site.git") // SSH remote
	sshOnly := env.account
	sshOnly.Alias = "ssh-user"
	env.deps.Accounts = &multiAccounts{primary: sshOnly, extra: nil}
	service := NewBindingService(env.deps)

	results, err := service.BindMany(ctx, []domain.AccountID{sshOnly.ID}, []string{"/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != "failed" {
		t.Fatalf("results = %+v", results)
	}
	if !strings.Contains(results[0].Reason, "SSH") || !strings.Contains(results[0].Reason, "账号") {
		t.Fatalf("reason must explain the mismatch plainly: %q", results[0].Reason)
	}
}

// multiAccounts serves one primary account plus extras.
type multiAccounts struct {
	primary domain.Account
	extra   []domain.Account
}

func (m *multiAccounts) Save(context.Context, domain.Account) error { return nil }
func (m *multiAccounts) Get(_ context.Context, id domain.AccountID) (domain.Account, error) {
	if m.primary.ID == id {
		return m.primary, nil
	}
	for _, account := range m.extra {
		if account.ID == id {
			return account, nil
		}
	}
	return domain.Account{}, domain.ErrAccountNotFound
}
func (m *multiAccounts) List(context.Context) ([]domain.Account, error) {
	return append([]domain.Account{m.primary}, m.extra...), nil
}
func (m *multiAccounts) Delete(context.Context, domain.AccountID) error { return nil }
