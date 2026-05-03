package github

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadAsset_WritesFileAndReportsProgress(t *testing.T) {
	body := strings.Repeat("X", 10_000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "v3.4.0/JetBrainsMono.zip")
		w.Header().Set("Content-Length", "10000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "JetBrainsMono.zip")
	var lastWritten, lastTotal int64
	var callCount int

	err := DownloadAsset(srv.URL+"/releases/download/v3.4.0/JetBrainsMono.zip", dest,
		func(written, total int64) {
			callCount++
			lastWritten = written
			lastTotal = total
		})
	require.NoError(t, err)

	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.Equal(t, int64(10000), info.Size())
	assert.Equal(t, int64(10000), lastWritten)
	assert.Equal(t, int64(10000), lastTotal)
	// Callback must fire at least once. The 10KB fixture body fits in one
	// io.Copy buffer (32KB) so multi-fire isn't structurally tested here;
	// real downloads of 10-50MB Nerd Font zips will fire many times.
	assert.GreaterOrEqual(t, callCount, 1, "onProgress should be called at least once")
}

func TestDownloadAsset_HTTPError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "x.zip")
	err := DownloadAsset(srv.URL+"/x.zip", dest, nil)
	assert.Error(t, err)

	_, statErr := os.Stat(dest)
	assert.True(t, errors.Is(statErr, os.ErrNotExist), "no partial file should be left behind")
}
