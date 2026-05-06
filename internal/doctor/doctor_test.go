package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Smoke: a freshly-seeded healthy environment should produce a result whose
// MaxSeverity is at most Warn (we accept Warn because the test box may
// or may not have fc-cache and may or may not be authenticated for GitHub).
func TestRun_Smoke_HealthyEnv(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(fontDir, 0o755))

	gh := github.NewClient()
	res, err := Run(Params{
		FontDir:     fontDir,
		StatePath:   filepath.Join(tmp, "state.json"),
		CatalogPath: filepath.Join(tmp, "catalog.json"),
		ArchivesDir: filepath.Join(tmp, "archives"),
		GitHub:      gh,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	sections := map[string]bool{}
	for _, c := range res.Checks {
		sections[c.Section] = true
	}
	for _, want := range []string{
		"XDG paths", "fc-cache", "GitHub auth", "Manifest",
		"Catalog cache", "Orphan directories",
	} {
		assert.True(t, sections[want], "missing section: %s", want)
	}

	assert.NotEqual(t, SeverityFail, res.MaxSeverity())
}

// Smoke: an actually-broken env (non-writable existing ancestor) yields at least one FAIL.
func TestRun_Smoke_BrokenPaths(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod restrictions are bypassed")
	}
	tmp := t.TempDir()
	locked := filepath.Join(tmp, "locked")
	require.NoError(t, os.MkdirAll(locked, 0o755))
	require.NoError(t, os.Chmod(locked, 0o500))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	bad := filepath.Join(locked, "deeper", "fonts")

	gh := github.NewClient()
	res, err := Run(Params{
		FontDir:     bad,
		StatePath:   filepath.Join(bad, "state.json"),
		CatalogPath: filepath.Join(bad, "catalog.json"),
		ArchivesDir: bad,
		GitHub:      gh,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, SeverityFail, res.MaxSeverity())
}
