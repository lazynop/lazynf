package doctor

import (
	"errors"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCatalog_Missing_Warn(t *testing.T) {
	checks := checkCatalog(nil, nil)
	require.Len(t, checks, 1)
	assert.Equal(t, SectionCatalog, checks[0].Section)
	assert.Equal(t, SeverityWarn, checks[0].Severity)
	assert.Contains(t, checks[0].Hint, "lazynf list")
}

func TestCheckCatalog_Fresh_OK(t *testing.T) {
	cat := &cache.Catalog{
		SchemaVersion: cache.CurrentSchemaVersion,
		Release:       "v3.4.0",
		CheckedAt:     time.Now().Add(-2 * time.Hour),
		Fonts:         []string{"FiraCode", "Hack", "JetBrainsMono"},
	}

	checks := checkCatalog(cat, nil)
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityOK, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "3 fonts")
}

func TestCheckCatalog_Stale_Warn(t *testing.T) {
	cat := &cache.Catalog{
		SchemaVersion: cache.CurrentSchemaVersion,
		Release:       "v3.4.0",
		CheckedAt:     time.Now().Add(-31 * 24 * time.Hour),
		Fonts:         []string{"FiraCode"},
	}

	checks := checkCatalog(cat, nil)
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityWarn, checks[0].Severity)
	assert.Contains(t, checks[0].Hint, "lazynf list")
}

func TestCheckCatalog_ParseError_Fail(t *testing.T) {
	checks := checkCatalog(nil, errors.New("invalid character"))
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityFail, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "parse")
}
