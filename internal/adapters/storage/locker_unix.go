//go:build unix

package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// FileLocker serializes write transactions across gitra processes using an
// advisory flock (baseline §31).
type FileLocker struct {
	path string
}

// NewFileLocker locks the given lock-file path.
func NewFileLocker(path string) *FileLocker { return &FileLocker{path: path} }

// WithWriteLock runs fn while holding an exclusive lock, honouring ctx
// cancellation for the wait.
func (l *FileLocker) WithWriteLock(ctx context.Context, fn func() error) error {
	if err := EnsureDir(filepath.Dir(l.path)); err != nil {
		return err
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
	defer func() { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) }()

	return fn()
}
