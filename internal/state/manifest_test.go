package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	m, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, 1, m.SchemaVersion)
	assert.Empty(t, m.Installed)
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	now := time.Date(2026, 5, 3, 11, 30, 0, 0, time.UTC)
	original := &Manifest{
		SchemaVersion: 1,
		Installed: map[string]InstalledFont{
			"JetBrainsMono": {
				Release:     "v3.4.0",
				InstalledAt: now,
				Dir:         "/u/alice/.local/share/fonts/JetBrainsMono",
				Files:       []string{"JetBrainsMonoNerdFont-Regular.ttf"},
			},
		},
	}
	require.NoError(t, original.Save(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}

func TestSave_AtomicViaTempRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	m := &Manifest{SchemaVersion: 1, Installed: map[string]InstalledFont{}}
	require.NoError(t, m.Save(path))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "state.json", entries[0].Name(), "no .tmp file should remain after Save")
}

func TestSave_CreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deep", "state.json")
	m := &Manifest{SchemaVersion: 1, Installed: map[string]InstalledFont{}}
	require.NoError(t, m.Save(path))
	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestLoad_CorruptJSON_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o644))
	_, err := Load(path)
	assert.Error(t, err)
}

func TestSave_PrettyPrintsForReadability(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	m := &Manifest{
		SchemaVersion: 1,
		Installed:     map[string]InstalledFont{},
	}
	require.NoError(t, m.Save(path))
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	// Confirm it's valid JSON and pretty-printed (has newlines).
	var anyOut any
	require.NoError(t, json.Unmarshal(content, &anyOut))
	assert.Contains(t, string(content), "\n")
}
