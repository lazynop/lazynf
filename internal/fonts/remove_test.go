package fonts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedRemoveState writes a manifest with the given installed entries.
func seedRemoveState(t *testing.T, path string, installed map[string]state.InstalledFont) {
	t.Helper()
	m := &state.Manifest{SchemaVersion: state.CurrentSchemaVersion, Installed: installed}
	require.NoError(t, m.Save(path))
}

// writeFontDir creates dir and writes each filename with a tiny payload.
func writeFontDir(t *testing.T, dir string, files []string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
	}
}

func TestRemove_InstalledFont_DeletesFilesAndManifest(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "FiraCode")
	files := []string{"FiraCode-Regular.ttf", "FiraCode-Bold.ttf"}
	writeFontDir(t, dir, files)

	seedRemoveState(t, statePath, map[string]state.InstalledFont{
		"FiraCode": {Release: "v3.4.0", Dir: dir, Files: files},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"FiraCode"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Removed)
	assert.Empty(t, res.Deadopted)
	assert.Empty(t, res.Failures)

	for _, f := range files {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.True(t, os.IsNotExist(err), "file %s should be gone", f)
	}
	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err), "dir should be gone")

	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.NotContains(t, m.Installed, "FiraCode")

	assert.True(t, fake.Called)
}

func TestRemove_NameNotInManifest_FailureRecorded(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	seedRemoveState(t, statePath, map[string]state.InstalledFont{}) // empty

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"GhostFont"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{})

	require.NoError(t, err)
	assert.Empty(t, res.Removed)
	assert.Empty(t, res.Deadopted)
	require.Contains(t, res.Failures, "GhostFont")
	assert.Contains(t, res.Failures["GhostFont"].Error(), "not installed")
	assert.False(t, fake.Called, "fc-cache must not run when nothing was deleted")
}

func TestRemove_ImportedFont_DefaultDeadopts(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "Hack")
	files := []string{"Hack-Regular.ttf"}
	writeFontDir(t, dir, files)

	seedRemoveState(t, statePath, map[string]state.InstalledFont{
		"Hack": {Release: state.ReleaseImported, Dir: dir, Files: files},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"Hack"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{})

	require.NoError(t, err)
	assert.Empty(t, res.Removed)
	assert.Equal(t, []string{"Hack"}, res.Deadopted)
	assert.Empty(t, res.Failures)

	// Files MUST still exist.
	_, err = os.Stat(filepath.Join(dir, files[0]))
	assert.NoError(t, err)
	_, err = os.Stat(dir)
	assert.NoError(t, err)

	// Manifest entry gone.
	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.NotContains(t, m.Installed, "Hack")

	// fc-cache NOT called: nothing was deleted.
	assert.False(t, fake.Called)
}

func TestRemove_NoManifestFile_AllFailures(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json") // does not exist

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"A", "B"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{})

	require.NoError(t, err)
	assert.Contains(t, res.Failures, "A")
	assert.Contains(t, res.Failures, "B")
	assert.False(t, fake.Called)
}

func TestRemove_ImportedFont_PurgeWithFiles_DeletesAll(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "Hack")
	files := []string{"Hack-Regular.ttf", "Hack-Bold.ttf"}
	writeFontDir(t, dir, files)

	seedRemoveState(t, statePath, map[string]state.InstalledFont{
		"Hack": {Release: state.ReleaseImported, Dir: dir, Files: files},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"Hack"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{Purge: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"Hack"}, res.Removed)
	assert.Empty(t, res.Deadopted)
	assert.Empty(t, res.Failures)

	for _, f := range files {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.True(t, os.IsNotExist(err))
	}
	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err))

	assert.True(t, fake.Called)
}
