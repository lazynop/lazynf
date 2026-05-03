package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestTag_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/ryanoasis/nerd-fonts/releases/latest", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":"v3.4.0","name":"v3.4.0"}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	tag, err := c.LatestTag("ryanoasis", "nerd-fonts")
	require.NoError(t, err)
	assert.Equal(t, "v3.4.0", tag)
}

func TestLatestTag_404_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	_, err := c.LatestTag("nope", "nope")
	assert.Error(t, err)
}

func TestLatestTag_EmptyTag_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":""}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	_, err := c.LatestTag("o", "r")
	assert.Error(t, err)
}
