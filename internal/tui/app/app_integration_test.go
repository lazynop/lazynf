package app

import (
	"net/http"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazynf/internal/engine"
	gh "github.com/lazynop/lazynf/internal/github"
	"github.com/lazynop/lazynf/internal/tui/messages"
)

// newDeadGitHubClient returns a *github.Client pointing at a non-routable
// address. Real requests fail fast with connection refused so the engine
// install goroutine can record a FailedEvent instead of nil-deref panicking.
func newDeadGitHubClient() *gh.Client {
	c := gh.NewClient()
	c.BaseURL = "http://127.0.0.1:1"
	c.HTTPClient = &http.Client{Timeout: 200 * time.Millisecond}
	return c
}

// TestIntegration_BootAndQuit drives the app through a realistic flow:
// window-size -> fonts-loaded -> cursor down -> quit.
func TestIntegration_BootAndQuit(t *testing.T) {
	eng := engine.New(engine.Deps{})
	m := New(eng)
	_ = m.Init()

	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 := out.(*Model)

	out, _ = m2.Update(messages.FontsLoadedMsg{
		Fonts: []engine.FontInfo{
			{Name: "FiraCode", Status: engine.StatusAvailable},
			{Name: "Hack", Status: engine.StatusAvailable},
			{Name: "Iosevka", Status: engine.StatusImported},
		},
	})
	m3 := out.(*Model)

	view := ansi.Strip(m3.View().Content)
	require.Contains(t, view, "FiraCode")
	require.Contains(t, view, "Hack")
	require.Contains(t, view, "Iosevka")

	out, _ = m3.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	out, cmd := out.(*Model).Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	require.NotNil(t, cmd)
	_, isQuit := cmd().(tea.QuitMsg)
	require.True(t, isQuit, "q should trigger tea.Quit when no ops in flight")
}

// TestIntegration_HelpOverlayToggle covers the most common in-session
// interaction: opening and closing the help overlay.
func TestIntegration_HelpOverlayToggle(t *testing.T) {
	eng := engine.New(engine.Deps{})
	m := New(eng)
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	out, _ = out.(*Model).Update(messages.FontsLoadedMsg{Fonts: []engine.FontInfo{{Name: "X"}}})

	// Open help.
	out, _ = out.(*Model).Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	mm := out.(*Model)
	require.Equal(t, OverlayHelp, mm.overlay)
	view := ansi.Strip(mm.View().Content)
	require.Contains(t, strings.ToLower(view), "key bindings")

	// Close help.
	out, _ = mm.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	require.Equal(t, OverlayNone, out.(*Model).overlay)
}

// TestIntegration_InstallRequestPipeline covers: cursor on a font, press 'i',
// app launches engine.Install and adds it to inFlight. We don't drain real
// events (no GitHub server here), just verify the request reached the app.
//
// The engine.Install goroutine will try to resolve the catalog and fail
// because no GitHub client is wired. We point the client at a non-routable
// address so the failure path stays in normal error returns (no nil-pointer
// panic).
func TestIntegration_InstallRequestPipeline(t *testing.T) {
	ghc := newDeadGitHubClient()
	eng := engine.New(engine.Deps{GitHub: ghc})
	m := New(eng)
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	out, _ = out.(*Model).Update(messages.FontsLoadedMsg{
		Fonts: []engine.FontInfo{{Name: "FiraCode", Status: engine.StatusAvailable}},
	})

	// Press 'i' -- fontlist emits RequestInstallMsg -> app picks it up.
	out, cmd := out.(*Model).Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	require.NotNil(t, cmd, "press i must produce a Cmd (RequestInstallMsg)")

	// Drain the cmd into the model so the install actually launches.
	msg := cmd()
	out, cmd = out.(*Model).Update(msg)
	mm := out.(*Model)
	require.NotEmpty(t, mm.inFlight, "an op should be tracked after install request")
	require.GreaterOrEqual(t, mm.statusbar.InFlight, 1)
	_ = cmd
}
