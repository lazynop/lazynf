package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/github"
	"github.com/lazynop/lazynf/internal/state"
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

func TestInstall_AlreadyImported_NoForce_EmitsConflictEvent(t *testing.T) {
	dir := t.TempDir()
	srv := newMockGitHubWithRelease(t, "v3.2.1", []string{"FiraCode"}, nil)
	t.Cleanup(srv.Close)
	gh := github.NewClient()
	gh.BaseURL = srv.URL

	// Seed manifest with FiraCode as "imported".
	statePath := filepath.Join(dir, "state.json")
	fontDirRoot := filepath.Join(dir, "fonts")
	require.NoError(t, os.MkdirAll(filepath.Join(fontDirRoot, "FiraCode"), 0o755))
	require.NoError(t, (&state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {
				Release:     state.ReleaseImported,
				InstalledAt: time.Now(),
				Dir:         filepath.Join(fontDirRoot, "FiraCode"),
				Files:       []string{"FiraCode-Regular.ttf"},
			},
		},
	}).Save(statePath))

	catPath := filepath.Join(dir, "catalog.json")
	require.NoError(t, (&cache.Catalog{
		Release:   "v3.2.1",
		Fonts:     []string{"FiraCode"},
		CheckedAt: time.Now(),
	}).Save(catPath))

	e := New(Deps{
		FontDir:     fontDirRoot,
		StatePath:   statePath,
		CatalogPath: catPath,
		GitHub:      gh,
		FontCache:   &fontcache.FakeRefresher{},
	})

	handle := e.Install(context.Background(), "FiraCode", InstallOptions{Force: false})

	var conflict *ConflictEvent
	var canceled bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range handle.Events {
			switch c := ev.(type) {
			case ConflictEvent:
				conflict = &c
				handle.Resolve(c.Token, ChoiceSkip)
			case CanceledEvent:
				canceled = true
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for handle.Events to close")
	}

	require.NotNil(t, conflict, "expected ConflictEvent")
	require.Equal(t, ConflictAlreadyImported, conflict.Kind)
	require.True(t, canceled, "expected CanceledEvent after ChoiceSkip")
}
