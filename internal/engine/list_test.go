package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/require"
)

func writeCatalogAt(t *testing.T, path string, c *cache.Catalog) {
	t.Helper()
	require.NoError(t, c.Save(path))
}

func writeManifestAt(t *testing.T, path string, m *state.Manifest) {
	t.Helper()
	require.NoError(t, m.Save(path))
}

func TestList_MixedStates(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.json")
	statePath := filepath.Join(dir, "state.json")
	fontDir := filepath.Join(dir, "fonts")

	cat := &cache.Catalog{
		Release:   "v3.2.1",
		Fonts:     []string{"FiraCode", "Hack", "JetBrainsMono", "Iosevka"},
		CheckedAt: time.Now(),
	}
	writeCatalogAt(t, catPath, cat)

	manifest := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {
				Release:     "v3.2.1",
				InstalledAt: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
				Dir:         filepath.Join(fontDir, "FiraCode"),
				Files:       []string{"FiraCode-Regular.ttf"},
			},
			"JetBrainsMono": {
				Release:     "v3.1.0", // stale
				InstalledAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				Dir:         filepath.Join(fontDir, "JetBrainsMono"),
				Files:       []string{"JetBrainsMono-Regular.ttf"},
			},
			"Iosevka": {
				Release:     state.ReleaseImported,
				InstalledAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				Dir:         filepath.Join(fontDir, "Iosevka"),
				Files:       []string{"Iosevka-Regular.ttf"},
			},
		},
	}
	writeManifestAt(t, statePath, manifest)

	e := New(Deps{
		FontDir:     fontDir,
		StatePath:   statePath,
		CatalogPath: catPath,
	})

	got, err := e.List(context.Background())
	require.NoError(t, err)

	byName := map[string]FontInfo{}
	for _, fi := range got {
		byName[fi.Name] = fi
	}

	require.Len(t, got, 4)
	require.Equal(t, StatusInstalled, byName["FiraCode"].Status)
	require.Equal(t, "v3.2.1", byName["FiraCode"].Version)
	require.Equal(t, "v3.2.1", byName["FiraCode"].LatestVersion)

	require.Equal(t, StatusStale, byName["JetBrainsMono"].Status)
	require.Equal(t, "v3.1.0", byName["JetBrainsMono"].Version)
	require.Equal(t, "v3.2.1", byName["JetBrainsMono"].LatestVersion)

	require.Equal(t, StatusImported, byName["Iosevka"].Status)
	require.Equal(t, "", byName["Iosevka"].Version)

	require.Equal(t, StatusAvailable, byName["Hack"].Status)
	require.Empty(t, byName["Hack"].Files)
}

func TestList_EmptyManifest(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.json")
	statePath := filepath.Join(dir, "state.json") // does not exist yet

	writeCatalogAt(t, catPath, &cache.Catalog{
		Release:   "v3.2.1",
		Fonts:     []string{"FiraCode"},
		CheckedAt: time.Now(),
	})

	e := New(Deps{StatePath: statePath, CatalogPath: catPath})
	got, err := e.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, StatusAvailable, got[0].Status)
}

func TestList_MissingCatalog_ReturnsManifestOnly(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.json") // does not exist
	statePath := filepath.Join(dir, "state.json")

	writeManifestAt(t, statePath, &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {Release: "v3.2.1", InstalledAt: time.Now(), Files: []string{"a.ttf"}},
		},
	})

	e := New(Deps{StatePath: statePath, CatalogPath: catPath})
	got, err := e.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, StatusUnknown, got[0].Status) // installed but no catalog → Unknown
	require.Equal(t, "v3.2.1", got[0].Version)
	require.Equal(t, "", got[0].LatestVersion)
}
