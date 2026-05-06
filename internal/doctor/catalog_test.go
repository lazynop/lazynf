package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCatalog_Missing_Warn(t *testing.T) {
	tmp := t.TempDir()
	checks := checkCatalog(filepath.Join(tmp, "catalog.json"))
	require.Len(t, checks, 1)
	assert.Equal(t, "Catalog cache", checks[0].Section)
	assert.Equal(t, SeverityWarn, checks[0].Severity)
	assert.Contains(t, checks[0].Hint, "lazynf list")
}

func TestCheckCatalog_Fresh_OK(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "catalog.json")
	c := &cache.Catalog{
		SchemaVersion: cache.CurrentSchemaVersion,
		Release:       "v3.4.0",
		CheckedAt:     time.Now().Add(-2 * time.Hour),
		Fonts:         []string{"FiraCode", "Hack", "JetBrainsMono"},
	}
	require.NoError(t, c.Save(path))

	checks := checkCatalog(path)
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityOK, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "3 fonts")
}

func TestCheckCatalog_Stale_Warn(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "catalog.json")
	c := &cache.Catalog{
		SchemaVersion: cache.CurrentSchemaVersion,
		Release:       "v3.4.0",
		CheckedAt:     time.Now().Add(-31 * 24 * time.Hour), // 31 days ago
		Fonts:         []string{"FiraCode"},
	}
	require.NoError(t, c.Save(path))

	checks := checkCatalog(path)
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityWarn, checks[0].Severity)
	assert.Contains(t, checks[0].Hint, "lazynf list")
}

func TestCheckCatalog_ParseError_Fail(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "catalog.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	checks := checkCatalog(path)
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityFail, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "parse")
}
