package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PickerPreferences struct {
	path string
}

type pickerSettings struct {
	StartDirectory string `json:"start_directory"`
}

func NewPickerPreferences(configDir string) *PickerPreferences {
	return &PickerPreferences{path: filepath.Join(configDir, "picker.json")}
}

func (p *PickerPreferences) StartDirectory() (string, error) {
	data, err := os.ReadFile(p.path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var settings pickerSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return "", err
	}
	return settings.StartDirectory, nil
}

func (p *PickerPreferences) SetStartDirectory(path string) error {
	data, err := json.MarshalIndent(pickerSettings{StartDirectory: path}, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(p.path, append(data, '\n'), 0o600)
}
