package fonts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanCatalogDirs_FontDirMissing_NilNoError(t *testing.T) {
	tmp := t.TempDir()
	out, err := ScanCatalogDirs(filepath.Join(tmp, "no-such"), []string{"FiraCode"})
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestScanCatalogDirs_OnlyCatalogMatches(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"FiraCode", "Hack", "MyCustomFont"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, name), 0o755))
	}
	// A non-dir file must be ignored.
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "Hack.txt"), []byte("x"), 0o644))

	out, err := ScanCatalogDirs(tmp, []string{"FiraCode", "Hack", "JetBrainsMono"})
	require.NoError(t, err)
	// Sorted alphabetically; MyCustomFont skipped (not in catalog).
	assert.Equal(t, []string{"FiraCode", "Hack"}, out)
}

func TestFindOrphans_FiltersInstalled(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"A", "B", "C"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, name), 0o755))
	}

	installed := map[string]state.InstalledFont{
		"A": {Release: "v3.4.0", Dir: filepath.Join(tmp, "A")},
	}
	out, err := FindOrphans(tmp, []string{"A", "B", "C", "D"}, installed)
	require.NoError(t, err)
	// A is installed → excluded. B and C are catalog dirs not in manifest → orphans.
	// D is in catalog but no dir on disk → not detected (correct).
	assert.Equal(t, []string{"B", "C"}, out)
}

func TestFindOrphans_FontDirMissing_NilNoError(t *testing.T) {
	tmp := t.TempDir()
	out, err := FindOrphans(filepath.Join(tmp, "no-such"), []string{"FiraCode"}, map[string]state.InstalledFont{})
	require.NoError(t, err)
	assert.Nil(t, out)
}
