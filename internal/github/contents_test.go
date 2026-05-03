package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchedFontsList_FiltersDirectoriesOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/ryanoasis/nerd-fonts/contents/patched-fonts", r.URL.Path)
		assert.Equal(t, "master", r.URL.Query().Get("ref"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"name":"0xProto","type":"dir"},
			{"name":"JetBrainsMono","type":"dir"},
			{"name":"README.md","type":"file"},
			{"name":"FiraCode","type":"dir"}
		]`))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	fonts, err := c.PatchedFontsList()
	require.NoError(t, err)
	assert.Equal(t, []string{"0xProto", "FiraCode", "JetBrainsMono"}, fonts)
}

func TestPatchedFontsList_404_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := NewClient()
	c.BaseURL = srv.URL
	_, err := c.PatchedFontsList()
	assert.Error(t, err)
}
