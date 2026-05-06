package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// catalogWith builds an in-memory catalog with the given font names.
func catalogWith(fonts []string) *cache.Catalog {
	return &cache.Catalog{
		SchemaVersion: cache.CurrentSchemaVersion,
		Release:       "v3.4.0",
		CheckedAt:     time.Now(),
		Fonts:         fonts,
	}
}

func TestCheckOrphans_NoCatalog_Warn(t *testing.T) {
	tmp := t.TempDir()
	checks := checkOrphans(filepath.Join(tmp, "fonts"), nil, nil)
	require.Len(t, checks, 1)
	assert.Equal(t, SectionOrphans, checks[0].Section)
	assert.Equal(t, SeverityWarn, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "catalog not cached")
}

func TestCheckOrphans_None(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(fontDir, 0o755))

	checks := checkOrphans(fontDir, nil, catalogWith([]string{"FiraCode", "Hack"}))
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityOK, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "none")
}

func TestCheckOrphans_DetectsUnmanagedDir(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(filepath.Join(fontDir, "FiraCode"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(fontDir, "Hack"), 0o755))

	// Only FiraCode is in the manifest; Hack is orphan.
	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {Release: "v3.4.0", Dir: filepath.Join(fontDir, "FiraCode")},
		},
	}

	checks := checkOrphans(fontDir, m, catalogWith([]string{"FiraCode", "Hack", "JetBrainsMono"}))
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityWarn, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "Hack")
	assert.NotContains(t, checks[0].Detail, "FiraCode")
	assert.Contains(t, checks[0].Hint, "lazynf import")
}

func TestCheckOrphans_IgnoresUnknownDirs(t *testing.T) {
	// A subdir whose name is not in the catalog must NOT be flagged
	// (could be a personal font outside the Nerd Fonts set).
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(filepath.Join(fontDir, "MyCustomFont"), 0o755))

	checks := checkOrphans(fontDir, nil, catalogWith([]string{"FiraCode", "Hack"}))
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityOK, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "none")
}
