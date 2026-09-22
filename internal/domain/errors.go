package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors follow the baseline error model (baseline §34).
var (
	// ErrInvalid matches any field-level invariant violation.
	ErrInvalid = errors.New("invalid domain value")

	ErrAccountNotFound = errors.New("account not found")
	ErrAccountExists   = errors.New("account already exists")
	ErrAccountInUse    = errors.New("account has active bindings")

	ErrNotGitRepository  = errors.New("not a git repository")
	ErrUnsupportedRepo   = errors.New("repository type is unsupported")
	ErrUnsupportedRemote = errors.New("remote transport is unsupported")
	ErrAlreadyBound      = errors.New("repository already bound")
	ErrBindingNotFound   = errors.New("binding not found")

	ErrAuthInvalid      = errors.New("authentication invalid")
	ErrProviderMismatch = errors.New("provider mismatch")
	ErrIdentityMismatch = errors.New("identity mismatch")

	ErrStateDrift = errors.New("managed state drift")
)

// ValidationError describes exactly one violated invariant.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}

// Is lets errors.Is(err, ErrInvalid) match every validation failure.
func (e *ValidationError) Is(target error) bool { return target == ErrInvalid }

func invalid(field, reason string) error {
	return &ValidationError{Field: field, Reason: reason}
}
