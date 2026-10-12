package app

// PickerPreferences is the boundary for the folder picker's saved starting directory.
type PickerPreferences interface {
	StartDirectory() (string, error)
	SetStartDirectory(path string) error
}
