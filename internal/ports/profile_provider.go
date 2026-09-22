package ports

import (
	"context"

	"github.com/zhanhd/gitra/internal/domain"
)

// ProviderProfile is the authenticated provider account used to auto-fill a
// gitra account after login (addendum §6.4).
type ProviderProfile struct {
	Provider      domain.ProviderType
	Host          string
	Username      string
	Name          string
	Email         string
	EmailFallback bool
}

// LoginToken is one acquired credential plus its source tag.
type LoginToken struct {
	Value  string
	Source string // explicit | env | stdin | gh | glab
}

// TokenProvider acquires a credential through the login ladder (addendum §4.2).
type TokenProvider interface {
	Acquire(ctx context.Context, providerType domain.ProviderType, explicit string, allowStdin, allowCLIReuse bool) (LoginToken, error)
}

// ProfileProvider fetches the authenticated provider profile for a token.
type ProfileProvider interface {
	Profile(ctx context.Context, providerType domain.ProviderType, host, token string) (ProviderProfile, error)
}
