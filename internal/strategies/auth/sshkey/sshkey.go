// Package sshkey implements the V1 ssh-key auth strategy (baseline §18.2).
package sshkey

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
	"github.com/zhanhd/gitra/internal/strategies/auth"
)

// ID is the strategy identifier used by Account.Transport.Strategy.
const ID = "ssh-key"

// Strategy builds core.sshCommand from a private key path.
type Strategy struct{}

// New returns the ssh-key strategy.
func New() *Strategy { return &Strategy{} }

// ID implements auth.Strategy.
func (s *Strategy) ID() string { return ID }

// Validate checks the account carries a usable private key path.
func (s *Strategy) Validate(ctx context.Context, account domain.Account) error {
	if account.Transport.Strategy != ID {
		return fmt.Errorf("%w: account %s uses transport %q", domain.ErrAuthInvalid, account.ID, account.Transport.Strategy)
	}
	if _, err := keyPath(account); err != nil {
		return err
	}
	return nil
}

// BuildGitConfig returns the single managed entry core.sshCommand. Identity
// keys (user.name/user.email) are composed by the binding service.
func (s *Strategy) BuildGitConfig(req auth.BuildRequest) ([]ports.GitConfigEntry, error) {
	path, err := keyPath(req.Account)
	if err != nil {
		return nil, err
	}
	command := fmt.Sprintf("ssh -i %q -o IdentitiesOnly=yes", path)
	if req.Account.Provider.Endpoint.SSHCommandNeedsPort(req.RemoteHasExplicitPort) {
		command += fmt.Sprintf(" -p %d", req.Account.Provider.Endpoint.SSHPort)
	}
	return []ports.GitConfigEntry{{Key: "core.sshCommand", Value: command}}, nil
}

func keyPath(account domain.Account) (string, error) {
	raw := strings.TrimSpace(account.Transport.Config["private_key"])
	if raw == "" {
		return "", fmt.Errorf("%w: account %s has no private_key", domain.ErrAuthInvalid, account.ID)
	}
	path, err := expandUser(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrAuthInvalid, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("%w: SSH private key does not exist: %s", domain.ErrAuthInvalid, raw)
	}
	return path, nil
}

func expandUser(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}
