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
		GitHub:      github.NewClient(),
		FontCache:   &fontcache.FakeRefresher{},
	})

	handle := e.Update(context.Background(), nil, UpdateOptions{})
	events := DrainEvents(t, handle)

	var failed []FailedEvent
	for _, ev := range events {
		if f, ok := ev.(FailedEvent); ok {
			failed = append(failed, f)
		}
	}
	require.Empty(t, failed, "no-installed update should not fail")
}
