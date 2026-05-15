package fonts

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectConflict_NewInstall_NoConflict(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "JetBrainsMono")
	m := &state.Manifest{Installed: map[string]state.InstalledFont{}}

	action, err := DetectConflict(m, "JetBrainsMono", dir, "v3.4.0", false)
	require.NoError(t, err)
	assert.Equal(t, ActionInstall, action)
}

func TestDetectConflict_lazynfManagedSameRelease_Skip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "JetBrainsMono")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	m := &state.Manifest{Installed: map[string]state.InstalledFont{
		"JetBrainsMono": {Release: "v3.4.0", Dir: dir},
	}}

	action, err := DetectConflict(m, "JetBrainsMono", dir, "v3.4.0", false)
	assert.True(t, errors.Is(err, ErrAlreadyInstalled))
	assert.Equal(t, ActionSkip, action)
}

func TestDetectConflict_lazynfManagedSameRelease_Force_Reinstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "JetBrainsMono")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	m := &state.Manifest{Installed: map[string]state.InstalledFont{
		"JetBrainsMono": {Release: "v3.4.0", Dir: dir},
	}}

	action, err := DetectConflict(m, "JetBrainsMono", dir, "v3.4.0", true)
	require.NoError(t, err)
	assert.Equal(t, ActionReinstall, action)
}

func TestDetectConflict_lazynfManagedDifferentRelease_Update(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "JetBrainsMono")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	m := &state.Manifest{Installed: map[string]state.InstalledFont{
		"JetBrainsMono": {Release: "v3.3.0", Dir: dir},
	}}

	action, err := DetectConflict(m, "JetBrainsMono", dir, "v3.4.0", false)
	require.NoError(t, err)
	assert.Equal(t, ActionReinstall, action)
}

func TestDetectConflict_DirExistsNotManaged_Errors(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "JetBrainsMono")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	m := &state.Manifest{Installed: map[string]state.InstalledFont{}}

	action, err := DetectConflict(m, "JetBrainsMono", dir, "v3.4.0", false)
	assert.True(t, errors.Is(err, ErrConflict))
	assert.Equal(t, ActionAbort, action)
}

func TestDetectConflict_DirExistsNotManaged_Force_Reinstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "JetBrainsMono")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	m := &state.Manifest{Installed: map[string]state.InstalledFont{}}

	action, err := DetectConflict(m, "JetBrainsMono", dir, "v3.4.0", true)
	require.NoError(t, err)
	assert.Equal(t, ActionReinstall, action)
}

func TestDetectConflict_Imported_NoForce_ReturnsConflictImported(t *testing.T) {
	dir := t.TempDir() // dir exists on disk
	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {
				Release:     state.ReleaseImported,
				InstalledAt: time.Now(),
				Dir:         dir,
				Files:       []string{"a.ttf"},
			},
		},
	}
	action, err := DetectConflict(m, "FiraCode", dir, "v3.2.1", false)
	require.Equal(t, ActionConflictImported, action)
	require.True(t, errors.Is(err, ErrAlreadyInstalled))
}

func TestDetectConflict_Imported_Force_ReturnsReinstall(t *testing.T) {
	dir := t.TempDir()
	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {
				Release:     state.ReleaseImported,
				InstalledAt: time.Now(),
				Dir:         dir,
				Files:       []string{"a.ttf"},
			},
		},
	}
	action, err := DetectConflict(m, "FiraCode", dir, "v3.2.1", true)
	require.Equal(t, ActionReinstall, action)
	require.NoError(t, err)
}

func TestDetectConflict_FilesOnDisk_NoForce_ReturnsAbort(t *testing.T) {
	dir := t.TempDir() // exists but not in manifest
	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed:     map[string]state.InstalledFont{},
	}
	action, err := DetectConflict(m, "Hack", dir, "v3.2.1", false)
	require.Equal(t, ActionAbort, action)
	require.True(t, errors.Is(err, ErrConflict))
}
