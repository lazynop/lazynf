package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/github"
	"github.com/stretchr/testify/require"
)

// TestImport_EmptyNames_NoAll_NoOp asserts that calling Import with no names
// and All=false is a no-op: no FailedEvent is emitted and the channel closes
// cleanly. A happy-path test requires real font files on disk and a fully
// wired catalog, deferred to cmd-level E2E coverage in Task 15.
func TestImport_EmptyNames_NoAll_NoOp(t *testing.T) {
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
	})

	handle := e.Import(context.Background(), nil, ImportOptions{})
	events := DrainEvents(t, handle)

	var failed []FailedEvent
	for _, ev := range events {
		if f, ok := ev.(FailedEvent); ok {
			failed = append(failed, f)
		}
	}
	require.Empty(t, failed, "no-op import should not fail")
}
