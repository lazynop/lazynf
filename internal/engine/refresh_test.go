package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/github"
	"github.com/stretchr/testify/require"
)

func TestRefreshCatalog_HappyPath(t *testing.T) {
	dir := t.TempDir()
	srv := newMockGitHubWithRelease(t, "v3.4.0", []string{"FiraCode", "Hack"}, nil)
	t.Cleanup(srv.Close)
	gh := github.NewClient()
	gh.BaseURL = srv.URL

	catPath := filepath.Join(dir, "catalog.json")
	e := New(Deps{
		CatalogPath: catPath,
		GitHub:      gh,
	})

	handle := e.RefreshCatalog(context.Background())
	events := DrainEvents(t, handle)

	var completed []CompletedEvent
	var failed []FailedEvent
	for _, ev := range events {
		switch e := ev.(type) {
		case CompletedEvent:
			completed = append(completed, e)
		case FailedEvent:
			failed = append(failed, e)
		}
	}
	require.Empty(t, failed)
	require.Len(t, completed, 1)

	// The catalog file must exist now.
	_, err := os.Stat(catPath)
	require.NoError(t, err)
	c, err := cache.Load(catPath)
	require.NoError(t, err)
	require.Equal(t, "v3.4.0", c.Release)
}

func TestRefreshCatalog_NetError_EmitsFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	gh := github.NewClient()
	gh.BaseURL = srv.URL

	dir := t.TempDir()
	e := New(Deps{
		CatalogPath: filepath.Join(dir, "catalog.json"),
		GitHub:      gh,
	})

	handle := e.RefreshCatalog(context.Background())
	events := DrainEvents(t, handle)

	var failed []FailedEvent
	for _, ev := range events {
		if f, ok := ev.(FailedEvent); ok {
			failed = append(failed, f)
		}
	}
	require.NotEmpty(t, failed)
}
