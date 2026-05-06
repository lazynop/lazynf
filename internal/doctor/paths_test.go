package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// happy path: all four dirs exist and are writable.
func TestCheckPaths_AllOK(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	stateDir := filepath.Join(tmp, "state")
	catDir := filepath.Join(tmp, "catalog")
	archDir := filepath.Join(tmp, "arch")
	for _, d := range []string{fontDir, stateDir, catDir, archDir} {
		require.NoError(t, os.MkdirAll(d, 0o755))
	}

	checks := checkPaths(Params{
		FontDir:     fontDir,
		StatePath:   filepath.Join(stateDir, "state.json"),
		CatalogPath: filepath.Join(catDir, "catalog.json"),
		ArchivesDir: archDir,
	})
	require.Len(t, checks, 4)
	for _, c := range checks {
		assert.Equal(t, SeverityOK, c.Severity, "check %q should be OK", c.Title)
		assert.Equal(t, "XDG paths", c.Section)
	}
}

// missing dir, parent writable -> WARN.
func TestCheckPaths_MissingButCreatable(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts") // not created

	checks := checkPaths(Params{
		FontDir:     fontDir,
		StatePath:   filepath.Join(tmp, "lazynf", "state.json"),
		CatalogPath: filepath.Join(tmp, "cache", "catalog.json"),
		ArchivesDir: filepath.Join(tmp, "cache", "archives"),
	})
	require.Len(t, checks, 4)
	for _, c := range checks {
		assert.Equal(t, SeverityWarn, c.Severity, "check %q should be WARN", c.Title)
		assert.Contains(t, c.Detail, "will be created")
	}
}

// First existing ancestor is non-writable -> FAIL (MkdirAll would error).
func TestCheckPaths_NonWritableAncestor_Fail(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod restrictions are bypassed")
	}
	tmp := t.TempDir()
	locked := filepath.Join(tmp, "locked")
	require.NoError(t, os.MkdirAll(locked, 0o755))
	require.NoError(t, os.Chmod(locked, 0o500))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) }) // restore so tmp cleanup can rm

	dir := filepath.Join(locked, "sub", "leaf")
	checks := checkPaths(Params{
		FontDir:     dir,
		StatePath:   filepath.Join(dir, "state.json"),
		CatalogPath: filepath.Join(dir, "catalog.json"),
		ArchivesDir: dir,
	})
	require.Len(t, checks, 4)
	for _, c := range checks {
		assert.Equal(t, SeverityFail, c.Severity, "check %q should be FAIL", c.Title)
	}
}
