package fonts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/github"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildDetectZip writes a zip whose .ttf entries match the given content map.
// Each entry in files is written verbatim. Uses python3 so we don't carry binary
// blobs in the repo (same pattern as buildSampleZip in install_test.go).
func buildDetectZip(t *testing.T, zipPath string, files map[string][]byte) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(zipPath), 0o755))

	// Write each font file to a temp dir; let Python read them back into the zip.
	// This avoids passing arbitrary bytes on the command line.
	dataDir := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, name), content, 0o644))
	}

	script := `
import sys, os, zipfile
out_path = sys.argv[1]
data_dir = sys.argv[2]
with zipfile.ZipFile(out_path, "w", compression=zipfile.ZIP_DEFLATED) as z:
    for fname in os.listdir(data_dir):
        full = os.path.join(data_dir, fname)
        with open(full, "rb") as f:
            z.writestr(fname, f.read())
    z.writestr("README.md", b"skip me")
`
	cmd := exec.Command("python3", "-c", script, zipPath, dataDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
}

// newImportMockServer builds an httptest.Server that serves:
//   - /repos/ryanoasis/nerd-fonts/releases/latest → tag
//   - /repos/ryanoasis/nerd-fonts/contents/patched-fonts → fonts list
//   - /releases/download/<tag>/<name>.zip → zip file at zipPaths[name]
func newImportMockServer(t *testing.T, tag string, fonts []string, zipPaths map[string]string) *httptest.Server {
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

// seedFontDir creates a font subdirectory with the given files (filename → content).
func seedFontDir(t *testing.T, fontDir, fontName string, files map[string][]byte) {
	t.Helper()
	dir := filepath.Join(fontDir, fontName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for name, data := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o644))
	}
}

// --- Tests ---

func TestImport_NamedFontWithoutDetect_ImportedSentinel(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")

	seedFontDir(t, fontDir, "FiraCode", map[string][]byte{
		"FiraCode-Regular.ttf": []byte("FAKE_TTF_R"),
		"FiraCode-Bold.ttf":    []byte("FAKE_TTF_B"),
	})

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode", "JetBrainsMono"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		Names:       []string{"FiraCode"},
		FontDir:     fontDir,
		StatePath:   statePath,
		CatalogPath: catPath,
		GitHub:      gh,
	}, ImportOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Imported)
	assert.Empty(t, res.Skipped)
	assert.Empty(t, res.Failures)
	assert.Equal(t, state.ReleaseImported, res.Details["FiraCode"])

	// Verify state was persisted correctly.
	m, err := state.Load(statePath)
	require.NoError(t, err)
	require.Contains(t, m.Installed, "FiraCode")
	entry := m.Installed["FiraCode"]
	assert.Equal(t, state.ReleaseImported, entry.Release)
	assert.ElementsMatch(t, []string{"FiraCode-Regular.ttf", "FiraCode-Bold.ttf"}, entry.Files)
	assert.Equal(t, filepath.Join(fontDir, "FiraCode"), entry.Dir)
}

func TestImport_DirNotFound_FailureRecorded(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(fontDir, 0o755))
	// Note: we do NOT seed FiraCode dir — it's missing.

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		Names:       []string{"FiraCode"},
		FontDir:     fontDir,
		StatePath:   filepath.Join(tmp, "state.json"),
		CatalogPath: filepath.Join(tmp, "catalog.json"),
		GitHub:      gh,
	}, ImportOptions{})

	require.NoError(t, err)
	assert.Empty(t, res.Imported)
	assert.Len(t, res.Failures, 1)
	assert.Error(t, res.Failures["FiraCode"])
}

func TestImport_FontNotInCatalog_FailureWithSuggestion(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	require.NoError(t, os.MkdirAll(fontDir, 0o755))

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode", "JetBrainsMono"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		Names:       []string{"NotAFont"},
		FontDir:     fontDir,
		StatePath:   filepath.Join(tmp, "state.json"),
		CatalogPath: filepath.Join(tmp, "catalog.json"),
		GitHub:      gh,
	}, ImportOptions{})

	require.NoError(t, err)
	assert.Empty(t, res.Imported)
	assert.Len(t, res.Failures, 1)
	assert.ErrorIs(t, res.Failures["NotAFont"], ErrFontNotFound)
}

func TestImport_AlreadyInState_NoForce_Skipped(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")

	seedFontDir(t, fontDir, "FiraCode", map[string][]byte{
		"FiraCode-Regular.ttf": []byte("FAKE_TTF_R"),
	})

	// Pre-populate state.
	m := &state.Manifest{SchemaVersion: 1, Installed: map[string]state.InstalledFont{
		"FiraCode": {
			Release: state.ReleaseImported,
			Dir:     filepath.Join(fontDir, "FiraCode"),
			Files:   []string{"FiraCode-Regular.ttf"},
		},
	}}
	require.NoError(t, m.Save(statePath))

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		Names:       []string{"FiraCode"},
		FontDir:     fontDir,
		StatePath:   statePath,
		CatalogPath: filepath.Join(tmp, "catalog.json"),
		GitHub:      gh,
		Force:       false,
	}, ImportOptions{})

	require.NoError(t, err)
	assert.Empty(t, res.Imported)
	assert.Equal(t, []string{"FiraCode"}, res.Skipped)
	assert.Empty(t, res.Failures)
}

func TestImport_AlreadyInState_Force_Reimported(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")

	seedFontDir(t, fontDir, "FiraCode", map[string][]byte{
		"FiraCode-Regular.ttf": []byte("FAKE_TTF_R"),
		"FiraCode-Bold.ttf":    []byte("FAKE_TTF_B"),
	})

	// Pre-populate state with stale file list.
	m := &state.Manifest{SchemaVersion: 1, Installed: map[string]state.InstalledFont{
		"FiraCode": {
			Release: state.ReleaseImported,
			Dir:     filepath.Join(fontDir, "FiraCode"),
			Files:   []string{"FiraCode-Regular.ttf"}, // missing Bold
		},
	}}
	require.NoError(t, m.Save(statePath))

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		Names:       []string{"FiraCode"},
		FontDir:     fontDir,
		StatePath:   statePath,
		CatalogPath: filepath.Join(tmp, "catalog.json"),
		GitHub:      gh,
		Force:       true,
	}, ImportOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Imported)
	assert.Empty(t, res.Skipped)
	assert.Empty(t, res.Failures)

	// State should be refreshed with both files.
	updated, err := state.Load(statePath)
	require.NoError(t, err)
	entry := updated.Installed["FiraCode"]
	assert.ElementsMatch(t, []string{"FiraCode-Regular.ttf", "FiraCode-Bold.ttf"}, entry.Files)
}

func TestImport_All_ScansDirAndImportsMatching(t *testing.T) {
	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")

	// 2 dirs match catalog, 1 doesn't.
	seedFontDir(t, fontDir, "FiraCode", map[string][]byte{"FiraCode-Regular.ttf": []byte("F")})
	seedFontDir(t, fontDir, "JetBrainsMono", map[string][]byte{"JetBrainsMono-Regular.ttf": []byte("J")})
	seedFontDir(t, fontDir, "NotACatalogFont", map[string][]byte{"some.ttf": []byte("N")})

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode", "JetBrainsMono"}, nil)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		All:         true,
		FontDir:     fontDir,
		StatePath:   filepath.Join(tmp, "state.json"),
		CatalogPath: filepath.Join(tmp, "catalog.json"),
		GitHub:      gh,
	}, ImportOptions{})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"FiraCode", "JetBrainsMono"}, res.Imported)
	assert.Empty(t, res.Skipped)
	assert.Empty(t, res.Failures)
}

func TestImport_DetectMode_HashesMatch_RealRelease(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}

	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")

	// Seed font dir with known content.
	localFiles := map[string][]byte{
		"FiraCode-Regular.ttf": []byte("MATCHING_BYTES_R"),
		"FiraCode-Bold.ttf":    []byte("MATCHING_BYTES_B"),
	}
	seedFontDir(t, fontDir, "FiraCode", localFiles)

	// Build a zip whose contents are identical to the on-disk files.
	zipPath := filepath.Join(tmp, "FiraCode.zip")
	buildDetectZip(t, zipPath, localFiles)

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode"}, map[string]string{"FiraCode": zipPath})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		Names:        []string{"FiraCode"},
		Detect:       true,
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
	}, ImportOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Imported)
	assert.Empty(t, res.Failures)
	// Hashes matched → real release tag.
	assert.Equal(t, "v3.4.0", res.Details["FiraCode"])

	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, "v3.4.0", m.Installed["FiraCode"].Release)
}

func TestImport_DetectMode_HashesDiffer_ImportedSentinel(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required to build zip fixture")
	}

	tmp := t.TempDir()
	fontDir := filepath.Join(tmp, "fonts")
	statePath := filepath.Join(tmp, "state.json")
	catPath := filepath.Join(tmp, "catalog.json")

	// On-disk files have different bytes than those served in the zip.
	seedFontDir(t, fontDir, "FiraCode", map[string][]byte{
		"FiraCode-Regular.ttf": []byte("LOCAL_DIFFERENT_R"),
		"FiraCode-Bold.ttf":    []byte("LOCAL_DIFFERENT_B"),
	})

	zipPath := filepath.Join(tmp, "FiraCode.zip")
	buildDetectZip(t, zipPath, map[string][]byte{
		"FiraCode-Regular.ttf": []byte("UPSTREAM_BYTES_R"),
		"FiraCode-Bold.ttf":    []byte("UPSTREAM_BYTES_B"),
	})

	srv := newImportMockServer(t, "v3.4.0", []string{"FiraCode"}, map[string]string{"FiraCode": zipPath})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL

	res, err := Import(context.Background(), ImportParams{
		Names:        []string{"FiraCode"},
		Detect:       true,
		FontDir:      fontDir,
		StatePath:    statePath,
		CatalogPath:  catPath,
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
	}, ImportOptions{})

	require.NoError(t, err)
	assert.Equal(t, []string{"FiraCode"}, res.Imported)
	assert.Empty(t, res.Failures)
	// Hashes differed → imported sentinel.
	assert.Equal(t, state.ReleaseImported, res.Details["FiraCode"])

	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, state.ReleaseImported, m.Installed["FiraCode"].Release)
}
