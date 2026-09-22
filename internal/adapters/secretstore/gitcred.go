// Package secretstore stores provider tokens in the platform credential store
// through git's native credential helpers. Tokens never touch gitra's JSON
// config, logs or .git/config (addendum §5.1).
package secretstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

var _ ports.SecretStore = (*GitCredentialStore)(nil)

// GitCredentialStore implements ports.SecretStore via git credential
// approve/fill/reject.
type GitCredentialStore struct {
	runner ports.CommandRunner
	helper string // when non-empty, force this helper (repo-local decision)
}

// NewGitCredentialStore builds the store; helper may be empty to reuse the
// user's configured helper chain.
func NewGitCredentialStore(runner ports.CommandRunner, helper string) *GitCredentialStore {
	return &GitCredentialStore{runner: runner, helper: helper}
}

// parseRef splits "<host>/<username>" references.
func parseRef(ref string) (host string, username string, err error) {
	host, username, ok := strings.Cut(ref, "/")
	if !ok || strings.TrimSpace(host) == "" || strings.TrimSpace(username) == "" {
		return "", "", fmt.Errorf("%w: credential reference must be <host>/<username>, got %q", domain.ErrAuthInvalid, ref)
	}
	return host, username, nil
}

func (s *GitCredentialStore) credentialRequest(host, username, password string) string {
	var builder strings.Builder
	builder.WriteString("protocol=https\n")
	builder.WriteString("host=" + host + "\n")
	builder.WriteString("username=" + username + "\n")
	if password != "" {
		builder.WriteString("password=" + password + "\n")
	}
	builder.WriteString("\n")
	return builder.String()
}

// credentialTimeout bounds a single credential-store operation. macOS may
// block on a Keychain prompt, and the UI must stay responsive.
const credentialTimeout = 15 * time.Second

func (s *GitCredentialStore) run(ctx context.Context, input string, args ...string) (ports.ProcessResult, error) {
	ctx, cancel := context.WithTimeout(ctx, credentialTimeout)
	defer cancel()

	full := make([]string, 0, len(args)+2)
	if s.helper != "" {
		full = append(full, "-c", "credential.helper="+s.helper)
	}
	full = append(full, args...)
	return s.runner.RunWithInput(ctx, "git", input, full...)
}

// Set implements ports.SecretStore.
func (s *GitCredentialStore) Set(ctx context.Context, ref string, secret string) error {
	host, username, err := parseRef(ref)
	if err != nil {
		return err
	}
	result, err := s.run(ctx, s.credentialRequest(host, username, secret), "credential", "approve")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("git credential approve failed: %s", strings.TrimSpace(result.Stderr))
	}
	return nil
}

// Get implements ports.SecretStore. A missing secret is ErrAuthInvalid.
func (s *GitCredentialStore) Get(ctx context.Context, ref string) (string, error) {
	host, username, err := parseRef(ref)
	if err != nil {
		return "", err
	}
	result, err := s.run(ctx, s.credentialRequest(host, username, ""), "credential", "fill")
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("%w: no stored credential for %s", domain.ErrAuthInvalid, ref)
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if password, ok := strings.CutPrefix(strings.TrimSpace(line), "password="); ok {
			if password == "" {
				break
			}
			return password, nil
		}
	}
	return "", fmt.Errorf("%w: credential for %s has no password", domain.ErrAuthInvalid, ref)
}

// Delete implements ports.SecretStore; deleting a missing entry is a no-op.
func (s *GitCredentialStore) Delete(ctx context.Context, ref string) error {
	host, username, err := parseRef(ref)
	if err != nil {
		return err
	}
	result, err := s.run(ctx, s.credentialRequest(host, username, ""), "credential", "reject")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 && result.ExitCode != 1 {
		return fmt.Errorf("git credential reject failed: %s", strings.TrimSpace(result.Stderr))
	}
	return nil
}
