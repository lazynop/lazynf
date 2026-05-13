package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/stretchr/testify/require"
)

func TestSearch_SubstringMatch(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.json")
	statePath := filepath.Join(dir, "state.json")

	writeCatalogAt(t, catPath, &cache.Catalog{
		Release:   "v3.2.1",
		Fonts:     []string{"FiraCode", "FiraMono", "Hack", "JetBrainsMono"},
		CheckedAt: time.Now(),
	})

	e := New(Deps{StatePath: statePath, CatalogPath: catPath})

	got, err := e.Search(context.Background(), "fira")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"FiraCode", "FiraMono"}, namesOf(got))

	got, err = e.Search(context.Background(), "Mono")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"FiraMono", "JetBrainsMono"}, namesOf(got))

	got, err = e.Search(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, got, 4) // empty query = no filter

	got, err = e.Search(context.Background(), "zzznope")
	require.NoError(t, err)
	require.Empty(t, got)
}

func namesOf(fis []FontInfo) []string {
	out := make([]string, 0, len(fis))
	for _, fi := range fis {
		out = append(out, fi.Name)
	}
	return out
}
