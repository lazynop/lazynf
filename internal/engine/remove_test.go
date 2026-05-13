package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/require"
)

// mkFontFiles is a test helper that creates a directory and writes a fake
// font file for each name under it. Returns the dir path on success.
func mkFontFiles(dir string, names ...string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("FAKE_TTF"), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func TestRemove_InstalledFont_EmitsCompletedSuccess(t *testing.T) {
	dir := t.TempDir()
	fontDir := filepath.Join(dir, "fonts", "FiraCode")
	require.NoError(t, mkFontFiles(fontDir, "Regular.ttf"))

	statePath := filepath.Join(dir, "state.json")
	require.NoError(t, (&state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"FiraCode": {
				Release:     "v3.2.1",
				InstalledAt: time.Now(),
				Dir:         fontDir,
				Files:       []string{"Regular.ttf"},
			},
		},
	}).Save(statePath))

	e := New(Deps{
		StatePath: statePath,
		FontCache: &fontcache.FakeRefresher{},
	})
	handle := e.Remove(context.Background(), []string{"FiraCode"}, RemoveOptions{})
	events := DrainEvents(t, handle)

	var completed []CompletedEvent
	for _, ev := range events {
		if c, ok := ev.(CompletedEvent); ok && c.Target == "FiraCode" {
			completed = append(completed, c)
		}
	}
	require.Len(t, completed, 1)
	require.Equal(t, CompletedSuccess, completed[0].Kind)
}

func TestRemove_ImportedFont_EmitsCompletedDeadopted(t *testing.T) {
	dir := t.TempDir()
	fontDir := filepath.Join(dir, "fonts", "Iosevka")
	require.NoError(t, mkFontFiles(fontDir, "Regular.ttf"))

	statePath := filepath.Join(dir, "state.json")
	require.NoError(t, (&state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed: map[string]state.InstalledFont{
			"Iosevka": {
				Release:     state.ReleaseImported,
				InstalledAt: time.Now(),
				Dir:         fontDir,
				Files:       []string{"Regular.ttf"},
			},
		},
	}).Save(statePath))

	e := New(Deps{StatePath: statePath, FontCache: &fontcache.FakeRefresher{}})
	handle := e.Remove(context.Background(), []string{"Iosevka"}, RemoveOptions{})
	events := DrainEvents(t, handle)

	var completed []CompletedEvent
	for _, ev := range events {
		if c, ok := ev.(CompletedEvent); ok && c.Target == "Iosevka" {
			completed = append(completed, c)
		}
	}
	require.Len(t, completed, 1)
	require.Equal(t, CompletedDeadopted, completed[0].Kind)
}

func TestRemove_NotInManifest_EmitsFailedForTag(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	e := New(Deps{StatePath: statePath, FontCache: &fontcache.FakeRefresher{}})
	handle := e.Remove(context.Background(), []string{"FiraCode"}, RemoveOptions{})
	events := DrainEvents(t, handle)

	var failed []FailedEvent
	for _, ev := range events {
		if f, ok := ev.(FailedEvent); ok && f.Target == "FiraCode" {
			failed = append(failed, f)
		}
	}
	require.GreaterOrEqual(t, len(failed), 1, "expected FailedEvent for non-existent font")
}
