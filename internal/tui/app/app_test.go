package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
)

func TestApp_BootsCleanly(t *testing.T) {
	eng := engine.New(engine.Deps{})
	m := New(eng)
	require.NotNil(t, m.Init())
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	out, _ = out.(*Model).Update(messages.FontsLoadedMsg{
		Fonts: []engine.FontInfo{{Name: "FiraCode"}},
	})
	_ = out.(*Model).View()
}

func TestApp_QuitImmediate_NoOpsInFlight(t *testing.T) {
	eng := engine.New(engine.Deps{})
	m := New(eng)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	require.NotNil(t, cmd)
	_, isQuit := cmd().(tea.QuitMsg)
	require.True(t, isQuit)
}

func TestApp_OpenCloseHelpOverlay(t *testing.T) {
	eng := engine.New(engine.Deps{})
	m := New(eng)
	out, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	require.Equal(t, OverlayHelp, out.(*Model).overlay)
	out, _ = out.(*Model).Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	require.Equal(t, OverlayNone, out.(*Model).overlay)
}
