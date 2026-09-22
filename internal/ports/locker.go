package ports

import "context"

// Locker serializes write operations across gitra processes (baseline §31).
// Callers hold the lock around complete write transactions, never around
// interactive sessions.
type Locker interface {
	WithWriteLock(ctx context.Context, fn func() error) error
}
