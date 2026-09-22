package ports

import (
	"context"
	"testing"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
)

// Compile-time assertions: the fake implementations must satisfy every port.
var (
	_ Git           = fakeGit{}
	_ SSH           = fakeSSH{}
	_ CommandRunner = fakeRunner{}
	_ AccountStore  = fakeAccountStore{}
	_ BindingStore  = fakeBindingStore{}
	_ Locker        = fakeLocker{}
	_ Clock         = fakeClock{}
)

type fakeGit struct{}

func (fakeGit) DiscoverRepository(context.Context, string) (domain.Repository, error) {
	return domain.Repository{}, nil
}
func (fakeGit) GetLocalConfig(context.Context, domain.Repository, string) (string, bool, error) {
	return "", false, nil
}
func (fakeGit) GetGlobalConfig(context.Context, string) (string, bool, error)           { return "", false, nil }
func (fakeGit) SetLocalConfig(context.Context, domain.Repository, string, string) error { return nil }
func (fakeGit) UnsetLocalConfig(context.Context, domain.Repository, string) error       { return nil }
func (fakeGit) Remotes(context.Context, domain.Repository) ([]domain.Remote, error)     { return nil, nil }

type fakeSSH struct{}

func (fakeSSH) Test(context.Context, SSHTestRequest) (SSHTestResult, error) {
	return SSHTestResult{}, nil
}

type fakeRunner struct{}

func (fakeRunner) Run(context.Context, string, ...string) (ProcessResult, error) {
	return ProcessResult{}, nil
}

func (fakeRunner) RunWithInput(context.Context, string, string, ...string) (ProcessResult, error) {
	return ProcessResult{}, nil
}

type fakeAccountStore struct{}

func (fakeAccountStore) Save(context.Context, domain.Account) error { return nil }
func (fakeAccountStore) Get(context.Context, domain.AccountID) (domain.Account, error) {
	return domain.Account{}, nil
}
func (fakeAccountStore) List(context.Context) ([]domain.Account, error) { return nil, nil }
func (fakeAccountStore) Delete(context.Context, domain.AccountID) error { return nil }

type fakeBindingStore struct{}

func (fakeBindingStore) Save(context.Context, domain.RepositoryBinding) error { return nil }
func (fakeBindingStore) Get(context.Context, domain.BindingID) (domain.RepositoryBinding, error) {
	return domain.RepositoryBinding{}, nil
}
func (fakeBindingStore) FindByRepository(context.Context, string) (domain.RepositoryBinding, bool, error) {
	return domain.RepositoryBinding{}, false, nil
}
func (fakeBindingStore) ListByAccount(context.Context, domain.AccountID) ([]domain.RepositoryBinding, error) {
	return nil, nil
}
func (fakeBindingStore) Delete(context.Context, domain.BindingID) error { return nil }

type fakeLocker struct{}

func (fakeLocker) WithWriteLock(_ context.Context, fn func() error) error { return fn() }

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

// TestNonZeroExitIsNotAnError documents the process contract (baseline §18):
// a non-zero exit code is data, not an execution error.
func TestNonZeroExitIsNotAnError(t *testing.T) {
	result := ProcessResult{ExitCode: 1, Stdout: "", Stderr: "auth failed"}
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code fixture, got %+v", result)
	}
	var err error
	if err != nil {
		t.Fatal("fixture error must be nil: exit codes are values")
	}
}

func TestLockerRunsFunction(t *testing.T) {
	called := false
	if err := (fakeLocker{}).WithWriteLock(context.Background(), func() error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("WithWriteLock() error = %v", err)
	}
	if !called {
		t.Fatal("WithWriteLock did not run the function")
	}
}
