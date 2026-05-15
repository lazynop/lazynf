// Package xdg resolves XDG Base Directory paths and lazynf-specific subpaths.
//
// On Linux, $XDG_DATA_HOME defaults to ~/.local/share and $XDG_CACHE_HOME defaults
// to ~/.cache. If $HOME is also unset, fall back to the OS temp dir to avoid
// returning empty paths.
package xdg

import (
	"os"
	"path/filepath"
)

const appName = "lazynf"

// DataHome returns the XDG data home directory.
func DataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "share")
	}
	return filepath.Join(os.TempDir(), appName, "share")
}

// CacheHome returns the XDG cache home directory.
func CacheHome() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return v
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".cache")
	}
	return filepath.Join(os.TempDir(), appName, "cache")
}

// StateHome returns the XDG state home directory.
func StateHome() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v
	}
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".local", "state")
	}
	return filepath.Join(os.TempDir(), appName, "state")
}

// lazynfDataDir is $XDG_DATA_HOME/lazynf.
func lazynfDataDir() string { return filepath.Join(DataHome(), appName) }

// lazynfCacheDir is $XDG_CACHE_HOME/lazynf.
func lazynfCacheDir() string { return filepath.Join(CacheHome(), appName) }

// DefaultFontDir is $XDG_DATA_HOME/fonts (per-user fontconfig location on Linux).
func DefaultFontDir() string { return filepath.Join(DataHome(), "fonts") }

// StateFile is the persistent manifest path.
func StateFile() string { return filepath.Join(lazynfDataDir(), "state.json") }

// CatalogFile is the regenerable catalog cache path.
func CatalogFile() string { return filepath.Join(lazynfCacheDir(), "catalog.json") }

// ArchivesDir is the optional kept-archives location (--keep-archive).
func ArchivesDir() string { return filepath.Join(lazynfCacheDir(), "archives") }
