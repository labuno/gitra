package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWiresServices(t *testing.T) {
	t.Setenv("GITRA_CONFIG_DIR", t.TempDir())

	application, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if application.Accounts == nil || application.Bindings == nil || application.Reconciler == nil {
		t.Fatalf("services not wired: %+v", application)
	}
	if application.Deps.Accounts == nil || application.Deps.Bindings == nil || application.Deps.Git == nil ||
		application.Deps.Snapshots == nil || application.Deps.Auth == nil || application.Deps.Routing == nil || application.Deps.Locker == nil {
		t.Fatalf("deps incomplete: %+v", application.Deps)
	}
	if err := application.Deps.Locker.WithWriteLock(context.Background(), func() error { return nil }); err != nil {
		t.Fatalf("locker unusable: %v", err)
	}

	second, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if second.Deps.Accounts == application.Deps.Accounts {
		t.Fatal("two New() calls must not share store instances")
	}
}

func TestNewFailsWhenConfigDirUnwritable(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	t.Setenv("GITRA_CONFIG_DIR", filepath.Join(parent, "nested"))
	if _, err := New(); err == nil {
		t.Fatal("New() must fail when the config dir cannot be created")
	}
}
