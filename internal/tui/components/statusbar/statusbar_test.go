package statusbar

import (
	"testing"

	tea "charm.land/bubbletea/v2"
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

func TestRender_ZeroWidth_UsesDefault(t *testing.T) {
	m := New(keys.Default())
	require.NotEmpty(t, m.View().Content)
}

func TestInit_ReturnsNil(t *testing.T) {
	require.Nil(t, New(keys.Default()).Init())
}

func TestUpdate_WindowSize_TracksWidth(t *testing.T) {
	m := New(keys.Default())
	out, cmd := m.Update(tea.WindowSizeMsg{Width: 123, Height: 24})
	require.Nil(t, cmd)
	require.Equal(t, 123, out.(Model).Width)
}

func TestUpdate_OtherMsg_IsNoOp(t *testing.T) {
	m := New(keys.Default())
	m.Width = 42
	out, cmd := m.Update("not a window size")
	require.Nil(t, cmd)
	require.Equal(t, 42, out.(Model).Width)
}
