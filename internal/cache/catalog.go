// Package cache stores lazynf's catalog cache: the list of available Nerd Fonts
// for a given upstream release tag. The cache is regenerable from GitHub —
// `lazynf cache clean` deletes it without losing anything important.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const CurrentSchemaVersion = 1

// Catalog is the on-disk cache shape.
type Catalog struct {
	SchemaVersion int       `json:"schema_version"`
	Release       string    `json:"release"`
	CheckedAt     time.Time `json:"checked_at"`
	Fonts         []string  `json:"fonts"`
}

// Load reads the catalog at the given path.
// Returns (nil, nil) if the file is missing — the caller is expected to refresh.
func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read catalog file: %w", err)
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse catalog file %s: %w", path, err)
	}
	return &c, nil
}

// IsFreshFor returns true if the cached catalog corresponds to the given upstream
// release tag — i.e., it does not need refreshing.
// Safe to call on a nil *Catalog (returns false).
func (c *Catalog) IsFreshFor(release string) bool {
	return c != nil && c.Release == release
}

// Save writes the catalog atomically.
func (c *Catalog) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp catalog: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename catalog file: %w", err)
	}
	return nil
}
