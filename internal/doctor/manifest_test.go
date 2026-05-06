package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckManifest_Missing_FirstRun(t *testing.T) {
	checks := checkManifest(false, nil, nil)
	require.Len(t, checks, 1)
	assert.Equal(t, SectionManifest, checks[0].Section)
	assert.Equal(t, SeverityOK, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "no manifest yet")
}

func TestCheckManifest_ParseError_Fail(t *testing.T) {
	checks := checkManifest(true, nil, errors.New("invalid character 'x'"))
	require.Len(t, checks, 1)
	assert.Equal(t, SeverityFail, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "parse error")
}

func TestCheckManifest_HappyPath_AllOnDisk(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "fonts", "FiraCode")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "FiraCode-Regular.ttf"), []byte("x"), 0o644))

	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {Release: "v3.4.0", Dir: dir, Files: []string{"FiraCode-Regular.ttf"}},
		},
	}

	checks := checkManifest(true, m, nil)
	require.NotEmpty(t, checks)
	assert.Equal(t, SectionManifest, checks[0].Section)
	for _, c := range checks {
		assert.NotEqual(t, SeverityFail, c.Severity)
		assert.NotEqual(t, SeverityWarn, c.Severity)
	}
}

func TestCheckManifest_DirMissing_Warn(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "fonts", "FiraCode") // never created

	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {Release: "v3.4.0", Dir: dir, Files: []string{"FiraCode-Regular.ttf"}},
		},
	}

	checks := checkManifest(true, m, nil)
	var sawWarn bool
	for _, c := range checks {
		if c.Severity == SeverityWarn {
			sawWarn = true
			assert.Contains(t, c.Detail, "FiraCode")
			assert.Contains(t, c.Detail, "dir missing")
		}
	}
	assert.True(t, sawWarn, "expected a WARN about FiraCode dir missing")
}

func TestCheckManifest_FileCountDiverges_Warn(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "fonts", "Hack")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// Manifest expects 2 files; only 1 on disk.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Hack-Regular.ttf"), []byte("x"), 0o644))

	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"Hack": {Release: "v3.4.0", Dir: dir, Files: []string{"Hack-Regular.ttf", "Hack-Bold.ttf"}},
		},
	}

	checks := checkManifest(true, m, nil)
	var sawWarn bool
	for _, c := range checks {
		if c.Severity == SeverityWarn {
			sawWarn = true
			assert.Contains(t, c.Detail, "Hack")
			assert.Contains(t, c.Detail, "expected 2")
			assert.Contains(t, c.Detail, "found 1")
		}
	}
	assert.True(t, sawWarn, "expected a WARN about Hack file count mismatch")
}

func TestCheckManifest_SchemaTooNew_Fail(t *testing.T) {
	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion + 1,
		Installed:     map[string]state.InstalledFont{},
	}

	checks := checkManifest(true, m, nil)
	require.NotEmpty(t, checks)
	assert.Equal(t, SeverityFail, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "schema")
}
