package app

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/fontcache"
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

// TestIntegration_RemoveConfirmDispatchesEngine guards against a past bug
// where the Remove confirm flow re-emitted RequestRemoveMsg instead of
// calling engine.Remove, looping the modal forever.
func TestIntegration_RemoveConfirmDispatchesEngine(t *testing.T) {
	// Wire a dead GitHub client so any net call is graceful (Remove is
	// local-only but state.Load still runs).
	deadGH := newDeadGitHubClient()

	eng := engine.New(engine.Deps{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
		GitHub:    deadGH,
		FontCache: &fontcache.FakeRefresher{},
	})
	m := New(eng)
	_ = m.Init()
	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Boot with one font in the list (we won't try to actually remove it
	// since the manifest is empty -- engine.Remove will emit a per-tag
	// FailedEvent, but the test only cares that an op was dispatched).
	out, _ = out.(*Model).Update(messages.FontsLoadedMsg{
		Fonts: []engine.FontInfo{
			{Name: "FiraCode", Status: engine.StatusInstalled, Version: "v3.2.1"},
		},
	})
	m2 := out.(*Model)

	// Press 'r' on the cursor row.
	out, cmd := m2.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	require.NotNil(t, cmd, "press r must produce a Cmd (RequestRemoveMsg)")

	// Pump the cmd through Update so app sees the RequestRemoveMsg.
	out, _ = out.(*Model).Update(cmd())
	m3 := out.(*Model)
	require.Equal(t, OverlayConfirm, m3.overlay, "expected confirm overlay after r")

	// Press 'y' to confirm. This should now dispatch engine.Remove (not
	// loop back to the modal).
	out, cmd = m3.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	require.NotNil(t, cmd, "press y in confirm must produce a Cmd")

	// Pump the ConfirmResultMsg through Update.
	out, cmd = out.(*Model).Update(cmd())
	m4 := out.(*Model)

	require.Equal(t, OverlayNone, m4.overlay, "confirm overlay must close on yes")
	require.NotEmpty(t, m4.inFlight, "engine.Remove should have launched and registered the op")
	require.GreaterOrEqual(t, m4.statusbar.InFlight, 1)
	_ = cmd
}
