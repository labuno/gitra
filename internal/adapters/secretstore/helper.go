package secretstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zhanhd/gitra/internal/ports"
)

// ResolveHelper decides which credential helper gitra should use:
//
//  1. the user's configured helper chain (returned as "" so git decides), else
//  2. Git Credential Manager on Windows (missing helper is an error), or
//  3. macOS osxkeychain when available, else a managed file store (0600).
func ResolveHelper(ctx context.Context, runner ports.CommandRunner, configDir string) (string, error) {
	// Explicit override (tests, portable installs, power users): use it as-is so
	// no test ever touches the real system keychain.
	if override := strings.TrimSpace(os.Getenv("GITRA_CREDENTIAL_HELPER")); override != "" {
		return override, nil
	}

	configured, err := runner.Run(ctx, "git", "config", "--get-all", "credential.helper")
	if err != nil {
		return "", err
	}
	if configured.ExitCode == 0 && strings.TrimSpace(configured.Stdout) != "" {
		return "", nil
	}
	if runtime.GOOS == "windows" {
		return windowsCredentialHelper(ctx, runner)
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

func windowsCredentialHelper(ctx context.Context, runner ports.CommandRunner) (string, error) {
	manager, err := runner.Run(ctx, "git", "credential-manager", "--version")
	if err != nil {
		return "", err
	}
	if manager.ExitCode != 0 {
		return "", fmt.Errorf("install Git for Windows with Git Credential Manager, or configure credential.helper")
	}
	return "manager", nil
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
	quoted := "'" + strings.ReplaceAll(filepath.ToSlash(storePath), "'", "'\"'\"'") + "'"
	return "store --file=" + quoted, nil
}
