package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withXDG points XDG_DATA_HOME and XDG_CACHE_HOME at t.TempDir() for the
// duration of the test, then returns the temp root.
func withXDG(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	return tmp
}

// seedCatalog writes a catalog.json under XDG_CACHE_HOME with the given font
// names. Caller must have set XDG_CACHE_HOME first (e.g. via withXDG).
func seedCatalog(t *testing.T, fonts []string) {
	t.Helper()
	c := &cache.Catalog{
		SchemaVersion: cache.CurrentSchemaVersion,
		Release:       "v3.4.0",
		CheckedAt:     time.Now(),
		Fonts:         fonts,
	}
	require.NoError(t, c.Save(filepath.Join(os.Getenv("XDG_CACHE_HOME"), "lazynf", "catalog.json")))
}

func TestCompleteFromCatalog_Missing_NoSuggestions(t *testing.T) {
	withXDG(t) // catalog not seeded

	out, _ := completeFromCatalog(nil, nil, "")
	assert.Empty(t, out)
}

func TestCompleteFromCatalog_ParseError_NoSuggestions(t *testing.T) {
	withXDG(t)
	catPath := filepath.Join(os.Getenv("XDG_CACHE_HOME"), "lazynf", "catalog.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(catPath), 0o755))
	require.NoError(t, os.WriteFile(catPath, []byte("not json"), 0o644))

	out, _ := completeFromCatalog(nil, nil, "")
	assert.Empty(t, out)
}

func TestCompleteFromCatalog_Populated_ReturnsAllNames(t *testing.T) {
	withXDG(t)
	seedCatalog(t, []string{"FiraCode", "Hack", "JetBrainsMono"})

	out, dir := completeFromCatalog(nil, nil, "")
	assert.ElementsMatch(t, []string{"FiraCode", "Hack", "JetBrainsMono"}, out)
	// Directive must disable file completion fallback so an empty/missing
	// catalog never accidentally suggests files in the cwd.
	assert.NotZero(t, dir, "expected ShellCompDirectiveNoFileComp")
}
