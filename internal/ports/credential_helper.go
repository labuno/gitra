package ports

import "context"

// CredentialHelperResolver decides which git credential helper gitra should
// write into a repository when the user has no global helper configured.
type CredentialHelperResolver interface {
	// Helper returns the helper spec to write; ok is false when the user
	// already has a configured helper chain and nothing must be written.
	Helper(ctx context.Context) (spec string, ok bool, err error)
}
