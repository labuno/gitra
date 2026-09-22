package ports

import (
	"context"

	"github.com/zhanhd/gitra/internal/domain"
)

// SSHIdentityParser interprets raw `ssh -T` output for a provider. The
// provider decides what "authenticated" means; business layers must not
// inspect exit codes directly (baseline §4.6).
type SSHIdentityParser interface {
	ParseSSHIdentity(ctx context.Context, providerType domain.ProviderType, stdout, stderr string) (username string, ok bool)
}
