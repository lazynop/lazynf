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
