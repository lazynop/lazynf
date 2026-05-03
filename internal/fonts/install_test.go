package fonts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lazynop/vellum/internal/cache"
	"github.com/lazynop/vellum/internal/fontcache"
	"github.com/lazynop/vellum/internal/github"
	"github.com/lazynop/vellum/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildSampleZip writes a tiny zip with two font files at p.
// Uses python3 instead of writing the binary inline so we don't ship a binary
// blob in the repo and don't risk an editor mangling line endings.
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

func newMockGitHubWithRelease(t *testing.T, tag string, fonts []string, zipPaths map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	})
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/contents/patched-fonts", func(w http.ResponseWriter, r *http.Request) {
		body := "["
		for i, f := range fonts {
			if i > 0 {
				body += ","
			}
			body += `{"name":"` + f + `","type":"dir"}`
		}
		body += "]"
		_, _ = w.Write([]byte(body))
	})
	for name, p := range zipPaths {
		// e.g. /releases/download/v3.4.0/JetBrainsMono.zip
		path := "/releases/download/" + tag + "/" + name + ".zip"
		fp := p
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, fp)
		})
	}
	return httptest.NewServer(mux)
}

func TestInstall_HappyPath_OneFont(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "JetBrainsMono.zip")
	buildSampleZip(t, zipPath, "JetBrainsMono")

	srv := newMockGitHubWithRelease(t, "v3.4.0",
		[]string{"FiraCode", "JetBrainsMono"},
		map[string]string{"JetBrainsMono": zipPath})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(fontDir, 0o755))

	fakeRefresher := &fontcache.FakeRefresher{}

	res, err := Install(context.Background(), InstallParams{
		Names:        []string{"JetBrainsMono"},
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		ArchivesDir:  filepath.Join(tmp, "archives"),
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    fakeRefresher,
	}, InstallOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"JetBrainsMono"}, res.Successes)
	assert.Empty(t, res.Failures)
	assert.True(t, fakeRefresher.Called)

	// State updated
	m, err := state.Load(statePath)
	require.NoError(t, err)
	require.Contains(t, m.Installed, "JetBrainsMono")
	assert.Equal(t, "v3.4.0", m.Installed["JetBrainsMono"].Release)
	assert.ElementsMatch(t,
		[]string{"JetBrainsMono-Regular.ttf", "JetBrainsMono-Bold.ttf"},
		m.Installed["JetBrainsMono"].Files)

	// Files extracted
	for _, f := range []string{"JetBrainsMono-Regular.ttf", "JetBrainsMono-Bold.ttf"} {
		_, err := os.Stat(filepath.Join(fontDir, "JetBrainsMono", f))
		assert.NoError(t, err)
	}

	// Catalog cached
	c, err := cache.Load(catPath)
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "v3.4.0", c.Release)
}

func TestInstall_UnknownFont_FailureRecorded(t *testing.T) {
	tmp := t.TempDir()
	srv := newMockGitHubWithRelease(t, "v3.4.0", []string{"FiraCode"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL
	res, err := Install(context.Background(), InstallParams{
		Names:        []string{"NotAFont"},
		FontDir:      filepath.Join(tmp, "fonts"),
		StatePath:    filepath.Join(tmp, "state.json"),
		CatalogPath:  filepath.Join(tmp, "catalog.json"),
		ArchivesDir:  filepath.Join(tmp, "archives"),
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, InstallOptions{})

	require.NoError(t, err) // batch returns nil even if individual font failed
	assert.Empty(t, res.Successes)
	assert.Len(t, res.Failures, 1)
	assert.ErrorIs(t, res.Failures["NotAFont"], ErrFontNotFound)
}

func TestInstall_BatchBestEffort_OneFailureOneSuccess(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "FiraCode.zip")
	buildSampleZip(t, zipPath, "FiraCode")

	// Catalog has both, but only FiraCode has a downloadable asset (JetBrains 404s)
	srv := newMockGitHubWithRelease(t, "v3.4.0",
		[]string{"FiraCode", "JetBrainsMono"},
		map[string]string{"FiraCode": zipPath})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(fontDir, 0o755))

	res, err := Install(context.Background(), InstallParams{
		Names:        []string{"FiraCode", "JetBrainsMono"},
		FontDir:      fontDir,
		StatePath:    filepath.Join(tmp, "state.json"),
		CatalogPath:  filepath.Join(tmp, "catalog.json"),
		ArchivesDir:  filepath.Join(tmp, "archives"),
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, InstallOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Successes)
	assert.Contains(t, res.Failures, "JetBrainsMono")
}

func TestInstall_SkipCacheRefresh_DoesNotCallRefresher(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "FiraCode.zip")
	buildSampleZip(t, zipPath, "FiraCode")

	srv := newMockGitHubWithRelease(t, "v3.4.0",
		[]string{"FiraCode"},
		map[string]string{"FiraCode": zipPath})
	defer srv.Close()
	gh := github.NewClient()
	gh.BaseURL = srv.URL

	fakeRefresher := &fontcache.FakeRefresher{}
	_, err := Install(context.Background(), InstallParams{
		Names:        []string{"FiraCode"},
		FontDir:      filepath.Join(tmp, "fonts"),
		StatePath:    filepath.Join(tmp, "state.json"),
		CatalogPath:  filepath.Join(tmp, "catalog.json"),
		ArchivesDir:  filepath.Join(tmp, "archives"),
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    fakeRefresher,
	}, InstallOptions{SkipCacheRefresh: true})
	require.NoError(t, err)
	assert.False(t, fakeRefresher.Called)
}

func TestInstall_AlreadyInstalledSameRelease_Skipped(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "FiraCode.zip")
	buildSampleZip(t, zipPath, "FiraCode")
	srv := newMockGitHubWithRelease(t, "v3.4.0",
		[]string{"FiraCode"},
		map[string]string{"FiraCode": zipPath})
	defer srv.Close()
	gh := github.NewClient()
	gh.BaseURL = srv.URL

	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")

	// Pre-populate state and dir.
	preDir := filepath.Join(fontDir, "FiraCode")
	require.NoError(t, os.MkdirAll(preDir, 0o755))
	pre := &state.Manifest{SchemaVersion: 1, Installed: map[string]state.InstalledFont{
		"FiraCode": {Release: "v3.4.0", Dir: preDir, Files: []string{"old.ttf"}},
	}}
	require.NoError(t, pre.Save(statePath))

	res, err := Install(context.Background(), InstallParams{
		Names:        []string{"FiraCode"},
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  filepath.Join(tmp, "catalog.json"),
		ArchivesDir:  filepath.Join(tmp, "archives"),
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, InstallOptions{})
	require.NoError(t, err)
	assert.Empty(t, res.Successes)
	assert.Equal(t, []string{"FiraCode"}, res.Skipped)
	assert.Empty(t, res.Failures)
}
