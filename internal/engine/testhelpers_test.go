package engine

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// buildSampleZip builds a small zip with two fake font files.
// Mirrors internal/fonts/install_test.go:buildSampleZip; uses python3 to
// avoid shipping a binary blob in the repo.
func buildSampleZip(t *testing.T, p, fontName string) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	cmd := exec.Command("python3", "-c",
		`import sys, zipfile
out, name = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(out, "w", compression=zipfile.ZIP_DEFLATED) as z:
    z.writestr(name+"-Regular.ttf", b"FAKE_TTF_R")
    z.writestr(name+"-Bold.ttf", b"FAKE_TTF_B")
    z.writestr("README.md", b"skip me")`,
		p, fontName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
}

// newMockGitHubWithRelease simulates the GitHub releases/contents endpoints
// that fonts.Install consumes. Mirrors internal/fonts/install_test.go.
func newMockGitHubWithRelease(t *testing.T, tag string, fontNames []string, zipPaths map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	})
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/contents/patched-fonts", func(w http.ResponseWriter, r *http.Request) {
		body := "["
		for i, f := range fontNames {
			if i > 0 {
				body += ","
			}
			body += `{"name":"` + f + `","type":"dir"}`
		}
		body += "]"
		_, _ = w.Write([]byte(body))
	})
	for name, p := range zipPaths {
		path := "/releases/download/" + tag + "/" + name + ".zip"
		fp := p
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, fp)
		})
	}
	return httptest.NewServer(mux)
}
