package fonts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/github"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUpdateMockServer mirrors newMockGitHubWithRelease but is scoped for update
// tests to avoid coupling to the install_test.go helper signature.
func newUpdateMockServer(t *testing.T, tag string, fonts []string, zipPaths map[string]string) *httptest.Server {
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
		path := "/releases/download/" + tag + "/" + name + ".zip"
		fp := p
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, fp)
		})
	}
	return httptest.NewServer(mux)
}

// buildUpdateZip creates a zip at p for fontName. Requires python3 (same guard
// as buildSampleZip in install_test.go).
func buildUpdateZip(t *testing.T, p, fontName string) {
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
    z.writestr(name+"-Bold.ttf",    b"FAKE_TTF_B")
    z.writestr("README.md",         b"skip me")`,
		p, fontName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
}

// seedState writes a state manifest with the given installed entries.
func seedState(t *testing.T, path string, installed map[string]state.InstalledFont) {
	t.Helper()
	m := &state.Manifest{SchemaVersion: 1, Installed: installed}
	require.NoError(t, m.Save(path))
}

// ---------- IsStale ----------

func TestIsStale_DifferentRelease(t *testing.T) {
	assert.True(t, IsStale("v3.3.0", "v3.4.0"))
}

func TestIsStale_SameRelease(t *testing.T) {
	assert.False(t, IsStale("v3.4.0", "v3.4.0"))
}

func TestIsStale_ImportedAlwaysStale(t *testing.T) {
	assert.True(t, IsStale(state.ReleaseImported, "v3.4.0"))
}

// ---------- Update ----------

// TestUpdate_NoArgs_AllStaleUpdated: state has 2 imported + 1 outdated + 1 fresh.
// No args → updates 3, leaves 1 fresh in AlreadyFresh.
func TestUpdate_NoArgs_AllStaleUpdated(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")
	tag := "v3.4.0"

	// Build zips for the three stale fonts.
	zips := map[string]string{}
	for _, name := range []string{"FiraCode", "JetBrainsMono", "Hack"} {
		p := filepath.Join(tmp, name+".zip")
		buildUpdateZip(t, p, name)
		zips[name] = p
	}

	allFonts := []string{"FiraCode", "JetBrainsMono", "Hack", "Inconsolata"}
	srv := newUpdateMockServer(t, tag, allFonts, zips)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	// Pre-seed font dirs so DetectConflict doesn't try to do a fresh install.
	for _, name := range allFonts {
		require.NoError(t, os.MkdirAll(filepath.Join(fontDir, name), 0o755))
	}

	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode":      {Release: state.ReleaseImported, Dir: filepath.Join(fontDir, "FiraCode")},
		"JetBrainsMono": {Release: state.ReleaseImported, Dir: filepath.Join(fontDir, "JetBrainsMono")},
		"Hack":          {Release: "v3.3.0", Dir: filepath.Join(fontDir, "Hack")},
		"Inconsolata":   {Release: tag, Dir: filepath.Join(fontDir, "Inconsolata")},
	})

	res, err := Update(context.Background(), UpdateParams{
		Names:        nil, // all installed
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"FiraCode", "JetBrainsMono", "Hack"}, res.Updated)
	assert.Equal(t, []string{"Inconsolata"}, res.AlreadyFresh)
	assert.Empty(t, res.Failures)
}

// TestUpdate_NoArgs_NothingStale_ReturnsAllFresh: all installed are at catalog.release.
func TestUpdate_NoArgs_NothingStale_ReturnsAllFresh(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")
	tag := "v3.4.0"

	allFonts := []string{"FiraCode", "JetBrainsMono"}
	srv := newUpdateMockServer(t, tag, allFonts, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	for _, name := range allFonts {
		require.NoError(t, os.MkdirAll(filepath.Join(fontDir, name), 0o755))
	}

	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode":      {Release: tag, Dir: filepath.Join(fontDir, "FiraCode")},
		"JetBrainsMono": {Release: tag, Dir: filepath.Join(fontDir, "JetBrainsMono")},
	})

	res, err := Update(context.Background(), UpdateParams{
		Names:        nil,
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{})

	require.NoError(t, err)
	assert.Empty(t, res.Updated)
	assert.ElementsMatch(t, []string{"FiraCode", "JetBrainsMono"}, res.AlreadyFresh)
	assert.Empty(t, res.Failures)
}

// TestUpdate_NamedFont_NotInState_FailureRecorded.
func TestUpdate_NamedFont_NotInState_FailureRecorded(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")

	srv := newUpdateMockServer(t, "v3.4.0", []string{"FiraCode"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	// Empty state: FiraCode is not installed.
	seedState(t, statePath, map[string]state.InstalledFont{})

	res, err := Update(context.Background(), UpdateParams{
		Names:        []string{"FiraCode"},
		FontDir:      filepath.Join(tmp, "fonts"),
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{})

	require.NoError(t, err)
	assert.Empty(t, res.Updated)
	assert.Len(t, res.Failures, 1)
	assert.Error(t, res.Failures["FiraCode"])
	assert.Contains(t, res.Failures["FiraCode"].Error(), "not installed")
}

// TestUpdate_NamedFont_AlreadyFresh_NotInUpdated_NoForce.
func TestUpdate_NamedFont_AlreadyFresh_NotInUpdated_NoForce(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")
	tag := "v3.4.0"

	srv := newUpdateMockServer(t, tag, []string{"FiraCode"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	require.NoError(t, os.MkdirAll(filepath.Join(fontDir, "FiraCode"), 0o755))
	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode": {Release: tag, Dir: filepath.Join(fontDir, "FiraCode")},
	})

	res, err := Update(context.Background(), UpdateParams{
		Names:        []string{"FiraCode"},
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{Force: false})

	require.NoError(t, err)
	assert.Empty(t, res.Updated)
	assert.Equal(t, []string{"FiraCode"}, res.AlreadyFresh)
	assert.Empty(t, res.Failures)
}

// TestUpdate_NamedFont_AlreadyFresh_Force_Updated.
func TestUpdate_NamedFont_AlreadyFresh_Force_Updated(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")
	tag := "v3.4.0"

	zipPath := filepath.Join(tmp, "FiraCode.zip")
	buildUpdateZip(t, zipPath, "FiraCode")

	srv := newUpdateMockServer(t, tag, []string{"FiraCode"}, map[string]string{"FiraCode": zipPath})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	require.NoError(t, os.MkdirAll(filepath.Join(fontDir, "FiraCode"), 0o755))
	seedState(t, statePath, map[string]state.InstalledFont{
		"FiraCode": {Release: tag, Dir: filepath.Join(fontDir, "FiraCode")},
	})

	res, err := Update(context.Background(), UpdateParams{
		Names:        []string{"FiraCode"},
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{Force: true, SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Updated)
	assert.Empty(t, res.AlreadyFresh)
	assert.Empty(t, res.Failures)
}

// TestUpdate_StaleFont_Updated_StateRefreshed: state at v3.3.0, catalog at v3.4.0.
func TestUpdate_StaleFont_Updated_StateRefreshed(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")

	zipPath := filepath.Join(tmp, "JetBrainsMono.zip")
	buildUpdateZip(t, zipPath, "JetBrainsMono")

	srv := newUpdateMockServer(t, "v3.4.0", []string{"JetBrainsMono"},
		map[string]string{"JetBrainsMono": zipPath})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	require.NoError(t, os.MkdirAll(filepath.Join(fontDir, "JetBrainsMono"), 0o755))
	seedState(t, statePath, map[string]state.InstalledFont{
		"JetBrainsMono": {Release: "v3.3.0", Dir: filepath.Join(fontDir, "JetBrainsMono")},
	})

	res, err := Update(context.Background(), UpdateParams{
		Names:        nil,
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"JetBrainsMono"}, res.Updated)
	assert.Empty(t, res.AlreadyFresh)
	assert.Empty(t, res.Failures)

	// State must reflect the new release tag.
	m, err := state.Load(statePath)
	require.NoError(t, err)
	require.Contains(t, m.Installed, "JetBrainsMono")
	assert.Equal(t, "v3.4.0", m.Installed["JetBrainsMono"].Release)
}

// TestUpdate_ImportedFont_AlwaysUpdated: release=imported → always stale without --force.
func TestUpdate_ImportedFont_AlwaysUpdated(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")

	zipPath := filepath.Join(tmp, "Hack.zip")
	buildUpdateZip(t, zipPath, "Hack")

	srv := newUpdateMockServer(t, "v3.4.0", []string{"Hack"},
		map[string]string{"Hack": zipPath})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	require.NoError(t, os.MkdirAll(filepath.Join(fontDir, "Hack"), 0o755))
	seedState(t, statePath, map[string]state.InstalledFont{
		"Hack": {Release: state.ReleaseImported, Dir: filepath.Join(fontDir, "Hack")},
	})

	res, err := Update(context.Background(), UpdateParams{
		Names:        nil,
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{Force: false, SkipCacheRefresh: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"Hack"}, res.Updated)
	assert.Empty(t, res.AlreadyFresh)
	assert.Empty(t, res.Failures)

	// State must now carry the real release tag, not "imported".
	m, err := state.Load(statePath)
	require.NoError(t, err)
	require.Contains(t, m.Installed, "Hack")
	assert.Equal(t, "v3.4.0", m.Installed["Hack"].Release)
}

// TestUpdate_NetworkErrorOnTag_PropagatesError: tag fetch failure → Update returns error.
func TestUpdate_NetworkErrorOnTag_PropagatesError(t *testing.T) {
	tmp := t.TempDir()

	// Server that always 500s.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	_, err := Update(context.Background(), UpdateParams{
		Names:        nil,
		FontDir:      filepath.Join(tmp, "fonts"),
		StatePath:    filepath.Join(tmp, "state.json"),
		CatalogPath:  filepath.Join(tmp, "catalog.json"),
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		Refresher:    &fontcache.FakeRefresher{},
	}, UpdateOptions{})

	assert.Error(t, err)
}
