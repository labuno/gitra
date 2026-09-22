package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDirOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITRA_CONFIG_DIR", dir)
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("ConfigDir() = %q, want %q", got, dir)
	}
}

func TestConfigDirDefaultUsesUserConfigDir(t *testing.T) {
	t.Setenv("GITRA_CONFIG_DIR", "")
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, filepath.Join("", "gitra")) && filepath.Base(got) != "gitra" {
		t.Fatalf("ConfigDir() = %q, want .../gitra", got)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bindings.json")
	if err := WriteFileAtomic(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("content = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want 600", perm)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("leftover temp files: %v", entries)
	}
}
