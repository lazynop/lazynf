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

func TestInstall_SingleFont_HappyPath(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "FiraCode.zip")
	buildSampleZip(t, zipPath, "FiraCode")

	srv := newMockGitHubWithRelease(t, "v3.2.1",
		[]string{"FiraCode"},
		map[string]string{"FiraCode": zipPath})
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
		FontDir:      filepath.Join(dir, "fonts"),
		StatePath:    filepath.Join(dir, "state.json"),
		CatalogPath:  catPath,
		ArchivesDir:  filepath.Join(dir, "archives"),
		GitHub:       gh,
		AssetURLBase: srv.URL + "/releases/download",
		FontCache:    &fontcache.FakeRefresher{},
	})

	handle := e.Install(context.Background(), "FiraCode", InstallOptions{})
	events := DrainEvents(t, handle)

	var (
		started, progress, completed int
		failed                       []FailedEvent
	)
	for _, ev := range events {
		switch e := ev.(type) {
		case StartedEvent:
			started++
		case ProgressEvent:
			progress++
		case CompletedEvent:
			completed++
			require.Equal(t, "FiraCode", e.Target)
			require.Equal(t, CompletedSuccess, e.Kind)
		case FailedEvent:
			failed = append(failed, e)
		}
	}
	require.GreaterOrEqual(t, started, 1)
	require.GreaterOrEqual(t, progress, 1, "expected ProgressEvent during download")
	require.Equal(t, 1, completed)
	require.Empty(t, failed)
}

func TestInstall_NotInCatalog_EmitsFailedForTag(t *testing.T) {
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

	handle := e.Install(context.Background(), "NopeFont", InstallOptions{})
	events := DrainEvents(t, handle)

	var failed []FailedEvent
	for _, ev := range events {
		if f, ok := ev.(FailedEvent); ok {
			failed = append(failed, f)
		}
	}
	require.GreaterOrEqual(t, len(failed), 1, "expected at least one FailedEvent for NopeFont")
	var hasTagFail bool
	for _, f := range failed {
		if f.Target == "NopeFont" {
			hasTagFail = true
			break
		}
	}
	require.True(t, hasTagFail, "expected FailedEvent.Target=NopeFont; got %#v", failed)
}
