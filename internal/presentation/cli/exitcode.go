package cli

import (
	"errors"

	"github.com/zhanhd/gitra/internal/domain"
)

// ExitCodeFor maps an error to the frozen exit-code contract (baseline §35).
func ExitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	switch {
	case errors.Is(err, errUsage), errors.Is(err, domain.ErrInvalid):
		return 2
	case errors.Is(err, domain.ErrAccountNotFound),
		errors.Is(err, domain.ErrAccountExists),
		errors.Is(err, domain.ErrAccountInUse):
		return 3
	case errors.Is(err, domain.ErrNotGitRepository):
		return 4
	case errors.Is(err, domain.ErrAuthInvalid),
		errors.Is(err, domain.ErrIdentityMismatch),
		errors.Is(err, domain.ErrProviderMismatch):
		return 5
	case errors.Is(err, domain.ErrAlreadyBound),
		errors.Is(err, domain.ErrBindingNotFound),
		errors.Is(err, domain.ErrStateDrift):
		return 6
	case errors.Is(err, domain.ErrUnsupportedRepo),
		errors.Is(err, domain.ErrUnsupportedRemote):
		return 7
	default:
		return 1
	}
}
