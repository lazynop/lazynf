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

	seedState(t, statePath, map[string]state.InstalledFont{
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
	seedState(t, statePath, map[string]state.InstalledFont{}) // empty

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

	seedState(t, statePath, map[string]state.InstalledFont{
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

func TestRemove_ImportedFont_PurgeNoFiles_FailureGuidesUser(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "Hack")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	seedState(t, statePath, map[string]state.InstalledFont{
		"Hack": {Release: state.ReleaseImported, Dir: dir, Files: nil},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"Hack"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{Purge: true})

	require.NoError(t, err)
	assert.Empty(t, res.Removed)
	assert.Empty(t, res.Deadopted)
	require.Contains(t, res.Failures, "Hack")
	msg := res.Failures["Hack"].Error()
	assert.Contains(t, msg, "no recorded files")
	assert.Contains(t, msg, "lazynf import")
	assert.Contains(t, msg, "--detect")

	// Manifest entry should NOT be removed.
	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.Contains(t, m.Installed, "Hack")

	// Dir untouched.
	_, err = os.Stat(dir)
	assert.NoError(t, err)

	assert.False(t, fake.Called)
}

func TestRemove_ImportedFont_PurgeWithFiles_DeletesAll(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "Hack")
	files := []string{"Hack-Regular.ttf", "Hack-Bold.ttf"}
	writeFontDir(t, dir, files)

	seedState(t, statePath, map[string]state.InstalledFont{
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

// Dir already gone (e.g. user did `rm -rf` themselves). Should still succeed
// and clean up the manifest.
func TestRemove_InstalledFont_DirAlreadyGone_AutoHeals(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "FiraCode") // never created
	files := []string{"FiraCode-Regular.ttf"}

	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode": {Release: "v3.4.0", Dir: dir, Files: files},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"FiraCode"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Removed)
	assert.Empty(t, res.Failures)

	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.NotContains(t, m.Installed, "FiraCode")
}

// Some files in Files exist, some don't. Should succeed; missing ones ignored.
func TestRemove_InstalledFont_PartialFilesMissing_Succeeds(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "FiraCode")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// Only write one of two recorded files.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "FiraCode-Regular.ttf"), []byte("x"), 0o644))

	files := []string{"FiraCode-Regular.ttf", "FiraCode-Bold.ttf"}
	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode": {Release: "v3.4.0", Dir: dir, Files: files},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"FiraCode"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Removed)
	assert.Empty(t, res.Failures)
	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

// User added an extra file in the font dir. Remove cleans the recorded files
// but leaves the dir (since it's still non-empty) and the extra file.
func TestRemove_InstalledFont_ExtraUserFile_DirAndFileLeft(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "FiraCode")
	files := []string{"FiraCode-Regular.ttf"}
	writeFontDir(t, dir, files)
	// User dropped a custom file in the dir.
	extra := filepath.Join(dir, "MY-NOTES.txt")
	require.NoError(t, os.WriteFile(extra, []byte("private"), 0o644))

	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode": {Release: "v3.4.0", Dir: dir, Files: files},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"FiraCode"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Removed)
	assert.Empty(t, res.Failures)

	// Recorded file gone, extra survives, dir survives.
	_, err = os.Stat(filepath.Join(dir, files[0]))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(extra)
	assert.NoError(t, err)
	_, err = os.Stat(dir)
	assert.NoError(t, err)

	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.NotContains(t, m.Installed, "FiraCode")
}

// One installed + one imported (deadopt) + one unknown — each populates the
// expected slot in the result.
func TestRemove_BatchMixed_SplitsResults(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	dirA := filepath.Join(tmp, "fonts", "Alpha")
	dirB := filepath.Join(tmp, "fonts", "Beta")
	writeFontDir(t, dirA, []string{"Alpha.ttf"})
	writeFontDir(t, dirB, []string{"Beta.ttf"})

	seedState(t, statePath, map[string]state.InstalledFont{
		"Alpha": {Release: "v3.4.0", Dir: dirA, Files: []string{"Alpha.ttf"}},
		"Beta":  {Release: state.ReleaseImported, Dir: dirB, Files: []string{"Beta.ttf"}},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"Alpha", "Beta", "Gamma"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"Alpha"}, res.Removed)
	assert.Equal(t, []string{"Beta"}, res.Deadopted)
	require.Contains(t, res.Failures, "Gamma")

	// Alpha files gone, Beta files left, Gamma untouched.
	_, err = os.Stat(filepath.Join(dirA, "Alpha.ttf"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dirB, "Beta.ttf"))
	assert.NoError(t, err)

	// fc-cache called once because Alpha was deleted.
	assert.True(t, fake.Called)
}

func TestRemove_SkipCacheRefresh_DoesNotInvokeRefresher(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	dir := filepath.Join(tmp, "fonts", "FiraCode")
	files := []string{"FiraCode-Regular.ttf"}
	writeFontDir(t, dir, files)

	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode": {Release: "v3.4.0", Dir: dir, Files: files},
	})

	fake := &fontcache.FakeRefresher{}
	res, err := Remove(context.Background(), RemoveParams{
		Names:     []string{"FiraCode"},
		StatePath: statePath,
		Refresher: fake,
	}, RemoveOptions{SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Removed)
	assert.False(t, fake.Called, "fc-cache must not run when SkipCacheRefresh is set")
}
