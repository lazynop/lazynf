package help

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/stretchr/testify/require"
)

func TestRender_ShowsAllKeyGroups(t *testing.T) {
	m := New(keys.Default())
	m.Width, m.Height = 80, 24
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "tab")
	require.Contains(t, s, "/")
	require.Contains(t, s, "i")
	require.Contains(t, s, "d")
	require.Contains(t, s, "q")
}

func TestRender_NarrowFallback(t *testing.T) {
	m := New(keys.Default())
	m.Width, m.Height = 40, 12
	_ = m.View()
}
