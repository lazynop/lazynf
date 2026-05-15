package logpane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileLogger_AppendsLines(t *testing.T) {
	dir := t.TempDir()
	l := NewFileLogger(dir)
	require.NoError(t, l.Write("first"))
	require.NoError(t, l.Write("second"))

	data, err := os.ReadFile(filepath.Join(dir, "lazynf", "tui.log"))
	require.NoError(t, err)
	s := string(data)
	require.Contains(t, s, "first")
	require.Contains(t, s, "second")
	require.Equal(t, 2, strings.Count(s, "\n"))
}

func TestFileLogger_RotatesOver1MiB(t *testing.T) {
	dir := t.TempDir()
	l := NewFileLogger(dir)
	path := filepath.Join(dir, "lazynf", "tui.log")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	big := strings.Repeat("x", 1<<20+1)
	require.NoError(t, os.WriteFile(path, []byte(big), 0o644))

	require.NoError(t, l.Write("after-rotation"))

	rot, err := os.Stat(path + ".0")
	require.NoError(t, err)
	require.Greater(t, rot.Size(), int64(1<<20))

	cur, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(cur), "after-rotation")
	require.Less(t, len(cur), 1<<19, "fresh log should be small")
}
