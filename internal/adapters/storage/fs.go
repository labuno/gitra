package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// schemaVersion is the persisted format version (baseline §30).
const schemaVersion = 1

// ConfigDir resolves the gitra config directory: GITRA_CONFIG_DIR wins
// (tests, portable installs), otherwise os.UserConfigDir()/gitra.
func ConfigDir() (string, error) {
	if dir := os.Getenv("GITRA_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "gitra"), nil
}

// EnsureDir creates dir (and parents) with owner-only permissions.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

// WriteFileAtomic writes data to path via write-temp + fsync + rename
// (baseline §29) so a crash never leaves a half-written JSON file.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
