// Package state manages lazynf's persistent manifest of installed fonts.
//
// The manifest is stored as JSON at $XDG_DATA_HOME/lazynf/state.json. It is
// NOT regenerable from other sources, so every write goes through an atomic
// temp+rename to avoid corruption on crash.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CurrentSchemaVersion is bumped whenever the on-disk format changes
// in a non-backward-compatible way.
const CurrentSchemaVersion = 1

// ReleaseImported is the sentinel value stored in InstalledFont.Release when a
// font was adopted via "lazynf import" without version detection. A future
// "lazynf update" will always refresh it because no real release tag matches
// this string.
const ReleaseImported = "imported"

// Manifest is the top-level on-disk structure.
type Manifest struct {
	SchemaVersion int                      `json:"schema_version"`
	Installed     map[string]InstalledFont `json:"installed"`
}

// InstalledFont records everything lazynf needs to manage a single installed font:
// which release it came from, when it was installed, where on disk it lives, and
// which files were extracted (so a future `lazynf remove` can clean up precisely).
type InstalledFont struct {
	Release     string    `json:"release"`
	InstalledAt time.Time `json:"installed_at"`
	Dir         string    `json:"dir"`
	Files       []string  `json:"files"`
}

// Load reads the manifest at the given path. If the file does not exist,
// returns a fresh empty manifest (no error).
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Manifest{SchemaVersion: CurrentSchemaVersion, Installed: map[string]InstalledFont{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse state file %s: %w", path, err)
	}
	if m.Installed == nil {
		m.Installed = map[string]InstalledFont{}
	}
	return &m, nil
}

// Save writes the manifest atomically: write to <path>.tmp, then rename.
// Creates parent directories as needed.
func (m *Manifest) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}
