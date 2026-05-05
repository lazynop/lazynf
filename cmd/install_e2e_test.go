package cmd_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/cmd"
	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E install: serves a fake catalog and a fake zip via httptest, runs the
// real install command, asserts state + filesystem are correct.
//
// We can't directly inject InstallParams from cmd_test, so we rely on env
// overrides: $XDG_DATA_HOME and $XDG_CACHE_HOME for paths, plus an
// internal hook (see cmd.SetTestAssetURLBase below) to point downloads
// at the test server.
func TestE2E_InstallOneFont(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "data", "fonts"), 0o755))

	// Build a fake font zip.
	zipPath := filepath.Join(tmp, "FiraCode.zip")
	cmdRun := exec.Command("python3", "-c",
		`import sys, zipfile
with zipfile.ZipFile(sys.argv[1], "w") as z:
    z.writestr("FiraCode-Regular.ttf", b"X")
    z.writestr("README.md", b"Y")`,
		zipPath)
	require.NoError(t, cmdRun.Run())

	// Mock GitHub.
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v3.4.0"}`))
	})
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/contents/patched-fonts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"FiraCode","type":"dir"}]`))
	})
	mux.HandleFunc("/releases/download/v3.4.0/FiraCode.zip", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cmd.SetTestGitHubBaseURL(srv.URL)
	cmd.SetTestAssetURLBase(srv.URL + "/releases/download")
	cmd.SetTestRefresher(&fontcache.FakeRefresher{})
	defer cmd.ResetTestOverrides()

	// Run `lazynf install FiraCode --no-cache-refresh`.
	root := cmd.NewRoot("test")
	root.SetArgs([]string{"install", "FiraCode", "--no-cache-refresh"})
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	err := root.Execute()
	require.NoError(t, err, "stderr=%s", stderr.String())

	// State should record FiraCode.
	m, err := state.Load(filepath.Join(tmp, "data", "lazynf", "state.json"))
	require.NoError(t, err)
	require.Contains(t, m.Installed, "FiraCode")
	assert.Equal(t, "v3.4.0", m.Installed["FiraCode"].Release)

	// Font file should be on disk.
	_, err = os.Stat(filepath.Join(tmp, "data", "fonts", "FiraCode", "FiraCode-Regular.ttf"))
	assert.NoError(t, err)
}
