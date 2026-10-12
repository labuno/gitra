//go:build windows

package storage

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestWindowsLockAcrossProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".lock")
	runChild := func(wantTimeout string) {
		child := exec.Command(os.Args[0], "-test.run=^TestWindowsLockChild$")
		child.Env = append(os.Environ(), "GITRA_TEST_LOCK="+path, "GITRA_TEST_TIMEOUT="+wantTimeout)
		if output, err := child.CombinedOutput(); err != nil {
			t.Fatalf("child lock: %v\n%s", err, output)
		}
	}
	if err := NewFileLocker(path).WithWriteLock(context.Background(), func() error { runChild("yes"); return nil }); err != nil {
		t.Fatal(err)
	}
	runChild("no")
}

func TestWindowsLockChild(t *testing.T) {
	path := os.Getenv("GITRA_TEST_LOCK")
	if path == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := NewFileLocker(path).WithWriteLock(ctx, func() error { return nil })
	if os.Getenv("GITRA_TEST_TIMEOUT") == "yes" {
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected contention timeout: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
}
