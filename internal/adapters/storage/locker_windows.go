//go:build windows

package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

type FileLocker struct {
	path string
}

func NewFileLocker(path string) *FileLocker { return &FileLocker{path: path} }

func (l *FileLocker) WithWriteLock(ctx context.Context, fn func() error) error {
	if err := EnsureDir(filepath.Dir(l.path)); err != nil {
		return err
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	handle := windows.Handle(file.Fd())
	var offset windows.Overlapped
	if err := acquireWindowsLock(ctx, handle, &offset); err != nil {
		return err
	}
	defer windows.UnlockFileEx(handle, 0, 1, 0, &offset)
	return fn()
}

func acquireWindowsLock(ctx context.Context, handle windows.Handle, offset *windows.Overlapped) error {
	for {
		err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, offset)
		if err == nil {
			return nil
		}
		if !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}
