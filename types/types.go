// Package types holds configuration shared across the helper's packages.
package types

// Config holds the filesystem locations a run operates on.
type Config struct {
	VaultPath     string
	TemplatesPath string
	NotesPath     string
}
