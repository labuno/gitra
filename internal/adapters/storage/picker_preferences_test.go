package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPickerPreferencesPersistStartDirectory(t *testing.T) {
	config := t.TempDir()
	store := NewPickerPreferences(config)
	if path, err := store.StartDirectory(); path != "" || err != nil {
		t.Fatalf("unset preference = %q, %v", path, err)
	}
	want := t.TempDir()
	if err := store.SetStartDirectory(want); err != nil {
		t.Fatal(err)
	}
	got, err := NewPickerPreferences(config).StartDirectory()
	if err != nil || got != want {
		t.Fatalf("saved preference = %q, %v", got, err)
	}
}

func TestPickerPreferencesReportReadAndWriteErrors(t *testing.T) {
	config := t.TempDir()
	if err := os.WriteFile(filepath.Join(config, "picker.json"), []byte("bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPickerPreferences(config).StartDirectory(); err == nil {
		t.Fatal("corrupt preferences must fail explicitly")
	}
	if err := NewPickerPreferences(filepath.Join(config, "picker.json")).SetStartDirectory(t.TempDir()); err == nil {
		t.Fatal("write failure must propagate")
	}
}
