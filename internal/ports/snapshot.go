package ports

import "github.com/zhanhd/gitra/internal/domain"

// ConfigState is the pre-bind state of one managed key (baseline §11).
type ConfigState struct {
	Exists bool
	Value  string
}

// Snapshot records the pre-bind state of every managed key so unbind can
// restore "the state before binding", not merely delete config.
type Snapshot struct {
	SchemaVersion int
	BindingID     domain.BindingID
	Previous      map[string]ConfigState
}

// SnapshotStore persists repository snapshots under <GitDir>/gitra/.
type SnapshotStore interface {
	Load(gitDir string) (Snapshot, bool, error)
	Save(gitDir string, snapshot Snapshot) error
	Delete(gitDir string) error
}
