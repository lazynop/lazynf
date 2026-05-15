package cmd_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lazynop/lazynf/cmd"
	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E: when state records FiraCode as imported and the user runs
// `lazynf install FiraCode` without --force, the engine emits a
// ConflictEvent. The CLI must auto-resolve with Skip, register a failure
// for FiraCode mentioning --force, and exit non-zero. The on-disk fonts
// must be left untouched.
func TestE2E_InstallConflict_AutoSkipWithForceHint(t *testing.T) {
	tmp := t.TempDir()
	dataHome := filepath.Join(tmp, "data")
	cacheHome := filepath.Join(tmp, "cache")
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)

	fontDir := filepath.Join(dataHome, "fonts")
	firaDir := filepath.Join(fontDir, "FiraCode")
	require.NoError(t, os.MkdirAll(firaDir, 0o755))
	firaFile := filepath.Join(firaDir, "FiraCode-Regular.ttf")
	require.NoError(t, os.WriteFile(firaFile, []byte("X"), 0o644))

	// Seed manifest with FiraCode as "imported" so the engine's pre-flight
	// raises ConflictAlreadyImported.
	statePath := filepath.Join(dataHome, "lazynf", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(statePath), 0o755))
	require.NoError(t, (&state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {
				Release:     state.ReleaseImported,
				InstalledAt: time.Now(),
				Dir:         firaDir,
				Files:       []string{"FiraCode-Regular.ttf"},
			},
		},
	}).Save(statePath))

	// Seed catalog so the engine doesn't try to refresh from the network
	// on its way to the conflict check (defensive; the conflict path
	// short-circuits before any release fetch, but we want a stable test).
	catalogPath := filepath.Join(cacheHome, "lazynf", "catalog.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(catalogPath), 0o755))
	require.NoError(t, (&cache.Catalog{
		Release:   "v3.4.0",
		Fonts:     []string{"FiraCode"},
		CheckedAt: time.Now(),
	}).Save(catalogPath))

	// Mock GitHub: never expected to be hit on the Skip path, but if it is
	// (e.g. future engine refactor), return a sane release to fail loudly
	// via assertion rather than via flaky network.
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v3.4.0"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cmd.SetTestGitHubBaseURL(srv.URL)
	cmd.SetTestAssetURLBase(srv.URL + "/releases/download")
	cmd.SetTestRefresher(&fontcache.FakeRefresher{})
	defer cmd.ResetTestOverrides()

	root := cmd.NewRoot("test")
	root.SetArgs([]string{"install", "FiraCode", "--no-cache-refresh"})
	var cobraOut, cobraErr bytes.Buffer
	root.SetOut(&cobraOut)
	root.SetErr(&cobraErr)

	// Verbosity writes directly to os.Stderr (Errorf/StyleFailure-rendered
	// failure lines), not via cobra's writer; redirect the real fd for the
	// duration of the call.
	out, err := captureStderr(t, func() error { return root.Execute() })
	require.Error(t, err, "expected non-zero exit on conflict")

	combined := cobraOut.String() + cobraErr.String() + out
	assert.Contains(t, strings.ToLower(combined), "conflict",
		"expected 'conflict' in output; got %q", combined)
	assert.Contains(t, combined, "--force",
		"expected '--force' hint in output; got %q", combined)

	// On-disk file must be untouched.
	b, statErr := os.ReadFile(firaFile)
	require.NoError(t, statErr)
	assert.Equal(t, []byte("X"), b, "font file must not be modified by a skipped conflict")

	// Manifest must still record FiraCode as imported.
	m, err := state.Load(statePath)
	require.NoError(t, err)
	require.Contains(t, m.Installed, "FiraCode")
	assert.Equal(t, state.ReleaseImported, m.Installed["FiraCode"].Release)
}

// captureStderr redirects os.Stderr to a pipe for the duration of fn,
// returning whatever was written. ui.Verbosity always wires its Errorf
// output to os.Stderr (via ui.New in cmd.Verbosity), so cobra's SetErr
// buffer alone is not enough to assert on user-facing error lines.
func captureStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, perr := os.Pipe()
	require.NoError(t, perr)
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	err := fn()
	_ = w.Close()
	os.Stderr = orig
	return <-done, err
}
