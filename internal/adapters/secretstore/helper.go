package secretstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhanhd/gitra/internal/ports"
)

// ResolveHelper decides which credential helper gitra should use:
//
//  1. the user's configured helper chain (returned as "" so git decides), else
//  2. the platform helper shipped with git (macOS osxkeychain), else
//  3. a gitra-managed file store under the config directory (0600).
func ResolveHelper(ctx context.Context, runner ports.CommandRunner, configDir string) (string, error) {
	configured, err := runner.Run(ctx, "git", "config", "--get-all", "credential.helper")
	if err != nil {
		return "", err
	}
	if configured.ExitCode == 0 && strings.TrimSpace(configured.Stdout) != "" {
		return "", nil
	}

	execPath, err := runner.Run(ctx, "git", "--exec-path")
	if err != nil {
		return "", err
	}
	if execPath.ExitCode == 0 {
		candidate := filepath.Join(strings.TrimSpace(execPath.Stdout), "git-credential-osxkeychain")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return "osxkeychain", nil
		}
	}

	return ManagedFileHelper(configDir)
}

// ManagedFileHelper returns the gitra-managed store helper and makes sure its
// directory and file permissions are owner-only.
func ManagedFileHelper(configDir string) (string, error) {
	storePath := filepath.Join(configDir, "credentials")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", err
	}
	if _, err := os.Stat(storePath); err == nil {
		if err := os.Chmod(storePath, 0o600); err != nil {
			return "", err
		}
	}
	return "store --file=" + storePath, nil
}
