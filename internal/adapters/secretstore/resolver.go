package secretstore

import (
	"context"

	"github.com/zhanhd/gitra/internal/ports"
)

var _ ports.CredentialHelperResolver = (*HelperResolver)(nil)

// HelperResolver adapts ResolveHelper to the ports.CredentialHelperResolver
// contract used by the binding service.
type HelperResolver struct {
	runner    ports.CommandRunner
	configDir string
}

// NewHelperResolver builds the resolver for one config directory.
func NewHelperResolver(runner ports.CommandRunner, configDir string) *HelperResolver {
	return &HelperResolver{runner: runner, configDir: configDir}
}

// Helper implements ports.CredentialHelperResolver.
func (h *HelperResolver) Helper(ctx context.Context) (string, bool, error) {
	spec, err := ResolveHelper(ctx, h.runner, h.configDir)
	if err != nil {
		return "", false, err
	}
	if spec == "" {
		return "", false, nil
	}
	return spec, true, nil
}
