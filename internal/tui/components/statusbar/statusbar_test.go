package statusbar

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazynf/internal/tui/keys"
)

func TestRender_ContainsCoreBindHints(t *testing.T) {
	m := New(keys.Default())
	m.Width = 80
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "?") // help bind
	require.Contains(t, s, "q") // quit bind
}

func TestRender_ShowsInFlightBadge(t *testing.T) {
	m := New(keys.Default())
	m.Width = 80
	m.InFlight = 3
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "3", "expected in-flight count")
	require.Contains(t, s, "ops")
}

func TestRender_ShowsSelectionBadge(t *testing.T) {
	m := New(keys.Default())
	m.Width = 80
	m.SelectionCount = 5
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "5")
	require.Contains(t, s, "selected")
}

func TestRender_NarrowWidthDoesNotPanic(t *testing.T) {
	m := New(keys.Default())
	m.Width = 20
	_ = m.View()
}
