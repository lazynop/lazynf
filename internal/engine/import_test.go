package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/github"
	"github.com/lazynop/lazynf/internal/state"
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

func TestImport_AlreadyImported_NoForce_EmitsConflictEvent(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	require.NoError(t, (&state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {Release: state.ReleaseImported, InstalledAt: time.Now()},
		},
	}).Save(statePath))

	e := New(Deps{
		FontDir:     filepath.Join(dir, "fonts"),
		StatePath:   statePath,
		CatalogPath: filepath.Join(dir, "catalog.json"),
	})

	handle := e.Import(context.Background(), []string{"FiraCode"}, ImportOptions{Force: false})

	var got *ConflictEvent
	var canceled bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range handle.Events {
			switch c := ev.(type) {
			case ConflictEvent:
				got = &c
				handle.Resolve(c.Token, ChoiceSkip)
			case CanceledEvent:
				canceled = true
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}

	require.NotNil(t, got, "expected ConflictEvent")
	require.Equal(t, ConflictAlreadyImported, got.Kind)
	require.True(t, canceled)
}
