package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/github"
	"github.com/stretchr/testify/require"
)

func TestUpdate_NoInstalled_Completes(t *testing.T) {
	dir := t.TempDir()
	srv := newMockGitHubWithRelease(t, "v3.2.1", []string{"FiraCode"}, nil)
	t.Cleanup(srv.Close)
	gh := github.NewClient()
	gh.BaseURL = srv.URL

	catPath := filepath.Join(dir, "catalog.json")
	statePath := filepath.Join(dir, "state.json")

	require.NoError(t, (&cache.Catalog{
		Release:   "v3.2.1",
		Fonts:     []string{"FiraCode"},
		CheckedAt: time.Now(),
	}).Save(catPath))

	e := New(Deps{
		FontDir:     filepath.Join(dir, "fonts"),
		StatePath:   statePath,
		CatalogPath: catPath,
		GitHub:      gh,
		FontCache:   &fontcache.FakeRefresher{},
	})

	handle := e.Update(context.Background(), nil, UpdateOptions{})
	events := DrainEvents(t, handle)

	// Assert StartedEvent and no FailedEvent for empty manifest.
	var failed []FailedEvent
	var started []StartedEvent
	for _, ev := range events {
		switch x := ev.(type) {
		case FailedEvent:
			failed = append(failed, x)
		case StartedEvent:
			started = append(started, x)
		}
	}
	require.Empty(t, failed)
	require.NotEmpty(t, started, "expected at least one StartedEvent")
	require.Equal(t, "update", started[0].Kind)
}

func TestUpdate_FontNotInstalled_EmitsFailedForTag(t *testing.T) {
	dir := t.TempDir()
	srv := newMockGitHubWithRelease(t, "v3.2.1", []string{"FiraCode"}, nil)
	t.Cleanup(srv.Close)
	gh := github.NewClient()
	gh.BaseURL = srv.URL

	catPath := filepath.Join(dir, "catalog.json")
	require.NoError(t, (&cache.Catalog{
		Release:   "v3.2.1",
		Fonts:     []string{"FiraCode"},
		CheckedAt: time.Now(),
	}).Save(catPath))

	e := New(Deps{
		FontDir:     filepath.Join(dir, "fonts"),
		StatePath:   filepath.Join(dir, "state.json"),
		CatalogPath: catPath,
		GitHub:      gh,
		FontCache:   &fontcache.FakeRefresher{},
	})

	// Try updating a font that is not installed.
	handle := e.Update(context.Background(), []string{"FiraCode"}, UpdateOptions{})
	events := DrainEvents(t, handle)

	var failed []FailedEvent
	for _, ev := range events {
		if f, ok := ev.(FailedEvent); ok {
			failed = append(failed, f)
		}
	}
	require.GreaterOrEqual(t, len(failed), 1, "expected at least one FailedEvent for non-installed font")
	var hasTagFail bool
	for _, f := range failed {
		if f.Target == "FiraCode" {
			hasTagFail = true
			break
		}
	}
	require.True(t, hasTagFail, "expected FailedEvent.Target=FiraCode; got %#v", failed)
}
