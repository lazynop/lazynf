package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingFile_ReturnsNil(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	cat, err := Load(path)
	require.NoError(t, err)
	assert.Nil(t, cat, "missing cache file means no cache yet — caller must refresh")
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	now := time.Date(2026, 5, 3, 11, 42, 0, 0, time.UTC)
	original := &Catalog{
		SchemaVersion: 1,
		Release:       "v3.4.0",
		CheckedAt:     now,
		Fonts:         []string{"0xProto", "JetBrainsMono", "ZedMono"},
	}
	require.NoError(t, original.Save(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}

func TestSave_AtomicViaTempRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.json")
	cat := &Catalog{SchemaVersion: 1, Release: "v1", Fonts: []string{}}
	require.NoError(t, cat.Save(path))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "catalog.json", entries[0].Name())
}

func TestSave_CreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "catalog.json")
	cat := &Catalog{SchemaVersion: 1, Release: "v1", Fonts: []string{}}
	require.NoError(t, cat.Save(path))
	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestIsFreshFor(t *testing.T) {
	cat := &Catalog{Release: "v3.4.0"}
	assert.True(t, cat.IsFreshFor("v3.4.0"))
	assert.False(t, cat.IsFreshFor("v3.5.0"))
}

func TestLoad_CorruptJSON_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	require.NoError(t, os.WriteFile(path, []byte("{bad"), 0o644))
	_, err := Load(path)
	assert.Error(t, err)
}
