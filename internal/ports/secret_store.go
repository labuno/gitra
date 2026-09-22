package ports

import "context"

// SecretStore persists provider tokens in the platform credential store.
// Values never travel through gitra's JSON config, logs or .git/config
// (addendum §5.1); references look like "<host>/<username>".
type SecretStore interface {
	Set(ctx context.Context, ref string, secret string) error
	Get(ctx context.Context, ref string) (string, error)
	Delete(ctx context.Context, ref string) error
}
