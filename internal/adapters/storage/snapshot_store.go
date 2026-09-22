package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zhanhd/gitra/internal/domain"
	"github.com/zhanhd/gitra/internal/ports"
)

// SnapshotStore implements ports.SnapshotStore with <GitDir>/gitra/state.json.
type SnapshotStore struct{}

// NewSnapshotStore returns the repository snapshot store.
func NewSnapshotStore() *SnapshotStore { return &SnapshotStore{} }

type configStateFile struct {
	Exists bool   `json:"exists"`
	Value  string `json:"value,omitempty"`
}

type snapshotFile struct {
	SchemaVersion int                        `json:"schema_version"`
	BindingID     string                     `json:"binding_id"`
	Previous      map[string]configStateFile `json:"previous"`
}

func snapshotPath(gitDir string) string {
	return filepath.Join(gitDir, "gitra", "state.json")
}

// Load reads the snapshot; ok is false when the repository has none.
func (s *SnapshotStore) Load(gitDir string) (ports.Snapshot, bool, error) {
	data, err := os.ReadFile(snapshotPath(gitDir))
	if err != nil {
		if os.IsNotExist(err) {
			return ports.Snapshot{}, false, nil
		}
		return ports.Snapshot{}, false, err
	}
	var file snapshotFile
	if err := json.Unmarshal(data, &file); err != nil {
		return ports.Snapshot{}, false, fmt.Errorf("%s: %w", snapshotPath(gitDir), err)
	}
	if file.SchemaVersion != schemaVersion {
		return ports.Snapshot{}, false, fmt.Errorf("unsupported snapshot schema_version %d", file.SchemaVersion)
	}
	snapshot := ports.Snapshot{
		SchemaVersion: file.SchemaVersion,
		BindingID:     domain.BindingID(file.BindingID),
		Previous:      map[string]ports.ConfigState{},
	}
	for key, state := range file.Previous {
		snapshot.Previous[key] = ports.ConfigState{Exists: state.Exists, Value: state.Value}
	}
	return snapshot, true, nil
}

// Save writes the snapshot atomically.
func (s *SnapshotStore) Save(gitDir string, snapshot ports.Snapshot) error {
	file := snapshotFile{
		SchemaVersion: schemaVersion,
		BindingID:     string(snapshot.BindingID),
		Previous:      map[string]configStateFile{},
	}
	for key, state := range snapshot.Previous {
		file.Previous[key] = configStateFile{Exists: state.Exists, Value: state.Value}
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(snapshotPath(gitDir), append(data, '\n'), 0o600)
}

// Delete removes the snapshot; deleting a missing snapshot is a no-op.
func (s *SnapshotStore) Delete(gitDir string) error {
	if err := os.Remove(snapshotPath(gitDir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
