package xdg

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataHome_FromEnv(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/data")
	t.Setenv("HOME", "/should/not/use")
	assert.Equal(t, "/custom/data", DataHome())
}

func TestDataHome_DefaultFromHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/u/alice")
	assert.Equal(t, "/u/alice/.local/share", DataHome())
}

func TestCacheHome_FromEnv(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/custom/cache")
	t.Setenv("HOME", "/should/not/use")
	assert.Equal(t, "/custom/cache", CacheHome())
}

func TestCacheHome_DefaultFromHome(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "/u/alice")
	assert.Equal(t, "/u/alice/.cache", CacheHome())
}

func TestLazynfDataDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/d")
	assert.Equal(t, filepath.Join("/d", "lazynf"), lazynfDataDir())
}

func TestLazynfCacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/c")
	assert.Equal(t, filepath.Join("/c", "lazynf"), lazynfCacheDir())
}

func TestDefaultFontDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/d")
	assert.Equal(t, filepath.Join("/d", "fonts"), DefaultFontDir())
}

func TestStateFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/d")
	assert.Equal(t, filepath.Join("/d", "lazynf", "state.json"), StateFile())
}

func TestCatalogFile(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/c")
	assert.Equal(t, filepath.Join("/c", "lazynf", "catalog.json"), CatalogFile())
}

func TestArchivesDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/c")
	assert.Equal(t, filepath.Join("/c", "lazynf", "archives"), ArchivesDir())
}

func TestFallback_NoHomeNoXDG(t *testing.T) {
	// On a stripped environment, we shouldn't crash. Keep a safe fallback.
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")
	assert.NotEmpty(t, DataHome())
}
