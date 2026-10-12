//go:build windows

package secretstore

import (
	"context"
	"strings"
	"testing"

	"github.com/zhanhd/gitra/internal/adapters/runner"
	"github.com/zhanhd/gitra/internal/ports"
)

func TestWindowsHelperUsesManagerWithoutGlobalConfiguration(t *testing.T) {
	r := &scriptedRunner{responses: map[string]ports.ProcessResult{
		"credential-manager --version": {ExitCode: 0, Stdout: "2.0"},
	}}
	helper, err := ResolveHelper(context.Background(), r, t.TempDir())
	if err != nil || helper != "manager" {
		t.Fatalf("default Windows helper = %q, %v", helper, err)
	}
}

func TestWindowsHelperReportsMissingManager(t *testing.T) {
	helper, err := ResolveHelper(context.Background(), &scriptedRunner{}, t.TempDir())
	if err == nil || helper != "" || !strings.Contains(err.Error(), "Git Credential Manager") {
		t.Fatalf("missing manager = %q, %v", helper, err)
	}
}

func TestWindowsCredentialManagerIsolatesAccounts(t *testing.T) {
	t.Setenv("GCM_INTERACTIVE", "never")
	t.Setenv("GCM_PROVIDER", "generic")
	t.Setenv("GCM_CREDENTIAL_STORE", "wincredman")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	store := NewGitCredentialStore(runner.New(), "manager")
	ctx := context.Background()
	probe, err := runner.New().RunWithInput(ctx, "git", "protocol=https\nhost=gitra-test.invalid\nusername=gitra-test-a\npassword=fixture-gitra-test-a\n\n", "credential-manager", "store")
	if err != nil || probe.ExitCode != 0 {
		t.Fatalf("Windows credential store probe: error=%v exit=%d stderr=%s", err, probe.ExitCode, probe.Stderr)
	}
	for _, user := range []string{"gitra-test-a", "gitra-test-b"} {
		ref := "gitra-test.invalid/" + user
		if err := store.Set(ctx, ref, "fixture-"+user); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = store.Delete(ctx, ref) })
	}
	for _, user := range []string{"gitra-test-a", "gitra-test-b"} {
		got, err := store.Get(ctx, "gitra-test.invalid/"+user)
		if err != nil || got != "fixture-"+user {
			t.Fatalf("credential isolation for %s failed: %v", user, err)
		}
	}
}
