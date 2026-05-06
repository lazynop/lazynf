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

// E2E: install a font through the real CLI, then remove it. Verifies that the
// remove path correctly reads the manifest written by install and cleans up
// disk + state.
func TestE2E_InstallThenRemove(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "data", "fonts"), 0o755))

	zipPath := filepath.Join(tmp, "FiraCode.zip")
	zipCmd := exec.Command("python3", "-c",
		`import sys, zipfile
with zipfile.ZipFile(sys.argv[1], "w") as z:
    z.writestr("FiraCode-Regular.ttf", b"X")
    z.writestr("README.md", b"Y")`,
		zipPath)
	require.NoError(t, zipCmd.Run())

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

	root := cmd.NewRoot("test")
	root.SetArgs([]string{"install", "FiraCode", "--no-cache-refresh"})
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	require.NoError(t, root.Execute(), "install stderr=%s", stderr.String())

	fontPath := filepath.Join(tmp, "data", "fonts", "FiraCode", "FiraCode-Regular.ttf")
	_, err := os.Stat(fontPath)
	require.NoError(t, err, "font file should exist after install")

	root2 := cmd.NewRoot("test")
	root2.SetArgs([]string{"remove", "FiraCode", "--no-cache-refresh"})
	stdout.Reset()
	stderr.Reset()
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	require.NoError(t, root2.Execute(), "remove stderr=%s", stderr.String())

	_, err = os.Stat(fontPath)
	assert.True(t, os.IsNotExist(err), "font file should be gone after remove")

	m, err := state.Load(filepath.Join(tmp, "data", "lazynf", "state.json"))
	require.NoError(t, err)
	assert.NotContains(t, m.Installed, "FiraCode")

	root3 := cmd.NewRoot("test")
	root3.SetArgs([]string{"remove", "FiraCode", "--no-cache-refresh"})
	stdout.Reset()
	stderr.Reset()
	root3.SetOut(&stdout)
	root3.SetErr(&stderr)
	err = root3.Execute()
	assert.Error(t, err, "remove of a non-installed font should error")
}
