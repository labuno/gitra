//go:build !unix

package storage

import (
	"context"
	"errors"
)

// ErrLockUnsupported is returned on platforms without file locking support.
var ErrLockUnsupported = errors.New("file locking is not supported on this platform")

// FileLocker is a stub on unsupported platforms (Windows is not a V1 target).
type FileLocker struct {
	path string
}

// NewFileLocker locks the given lock-file path.
func NewFileLocker(path string) *FileLocker { return &FileLocker{path: path} }

// WithWriteLock fails closed on unsupported platforms.
func (l *FileLocker) WithWriteLock(ctx context.Context, fn func() error) error {
	return ErrLockUnsupported
}
