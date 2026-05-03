package fonts

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/vellum/internal/cache"
	"github.com/lazynop/vellum/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockGitHub(t *testing.T, tag string, fonts []string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	})
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/contents/patched-fonts", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "master" {
			t.Errorf("expected ?ref=master, got %q", r.URL.Query().Get("ref"))
		}
		w.WriteHeader(http.StatusOK)
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
	return httptest.NewServer(mux)
}

func TestResolveCatalog_NoCache_FetchesAndPersists(t *testing.T) {
	srv := newMockGitHub(t, "v3.4.0", []string{"FiraCode", "JetBrainsMono"})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL
	catPath := filepath.Join(t.TempDir(), "catalog.json")

	cat, err := ResolveCatalog(gh, catPath)
	require.NoError(t, err)
	assert.Equal(t, "v3.4.0", cat.Release)
	assert.Equal(t, []string{"FiraCode", "JetBrainsMono"}, cat.Fonts)

	// And it should be persisted.
	loaded, err := cache.Load(catPath)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "v3.4.0", loaded.Release)
}

func TestResolveCatalog_CachedAndFresh_NoSecondFetch(t *testing.T) {
	tagCalls := 0
	contentsCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		tagCalls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":"v3.4.0"}`))
	})
	mux.HandleFunc("/repos/ryanoasis/nerd-fonts/contents/patched-fonts", func(w http.ResponseWriter, r *http.Request) {
		contentsCalls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name":"X","type":"dir"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL
	catPath := filepath.Join(t.TempDir(), "catalog.json")

	// Pre-seed cache with same tag.
	pre := &cache.Catalog{SchemaVersion: 1, Release: "v3.4.0", Fonts: []string{"X"}}
	require.NoError(t, pre.Save(catPath))

	_, err := ResolveCatalog(gh, catPath)
	require.NoError(t, err)
	assert.Equal(t, 1, tagCalls, "tag should be checked")
	assert.Equal(t, 0, contentsCalls, "contents should NOT be re-fetched when cache matches")
}

func TestResolveCatalog_CachedButStale_Refetches(t *testing.T) {
	srv := newMockGitHub(t, "v3.4.0", []string{"NewFont"})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL
	catPath := filepath.Join(t.TempDir(), "catalog.json")

	pre := &cache.Catalog{SchemaVersion: 1, Release: "v3.3.0", Fonts: []string{"OldFont"}}
	require.NoError(t, pre.Save(catPath))

	cat, err := ResolveCatalog(gh, catPath)
	require.NoError(t, err)
	assert.Equal(t, "v3.4.0", cat.Release)
	assert.Equal(t, []string{"NewFont"}, cat.Fonts)
}

func TestResolveCatalog_CorruptCache_RefetchesAndOverwrites(t *testing.T) {
	srv := newMockGitHub(t, "v3.4.0", []string{"FiraCode"})
	defer srv.Close()

	gh := github.NewClient()
	gh.BaseURL = srv.URL
	catPath := filepath.Join(t.TempDir(), "catalog.json")

	// Write a corrupt cache file.
	require.NoError(t, os.WriteFile(catPath, []byte("{not json"), 0o644))

	cat, err := ResolveCatalog(gh, catPath)
	require.NoError(t, err, "corrupt cache must self-heal, not fail")
	assert.Equal(t, "v3.4.0", cat.Release)
	assert.Equal(t, []string{"FiraCode"}, cat.Fonts)

	// Confirm overwrite happened.
	loaded, err := cache.Load(catPath)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "v3.4.0", loaded.Release)
}
