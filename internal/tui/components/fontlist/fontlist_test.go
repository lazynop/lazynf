package fontlist

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/stretchr/testify/require"
)

func makeFonts() []engine.FontInfo {
	return []engine.FontInfo{
		{Name: "FiraCode", Status: engine.StatusInstalled, Version: "v3.2.1"},
		{Name: "Hack", Status: engine.StatusAvailable, LatestVersion: "v3.2.1"},
		{Name: "Iosevka", Status: engine.StatusImported},
		{Name: "JetBrainsMono", Status: engine.StatusStale, Version: "v3.0.0", LatestVersion: "v3.2.1"},
	}
}

func TestInit_LoadsFonts(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	require.Len(t, m.fonts, 4)
}

func TestDown_MovesCursor(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	require.Equal(t, 0, m.cursor)
	m, _ = applyMsg(m, keyPress("j"))
	require.Equal(t, 1, m.cursor)
}

func TestSpace_TogglesSelection(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m, cmd := applyMsg(m, keyPress(" "))
	require.True(t, m.selected["FiraCode"])
	require.NotNil(t, cmd)
	got := cmd().(messages.SelectionChangedMsg)
	require.Equal(t, 1, got.Count)

	m, cmd = applyMsg(m, keyPress(" "))
	require.False(t, m.selected["FiraCode"])
	require.Equal(t, 0, cmd().(messages.SelectionChangedMsg).Count)
}

func TestInstallKey_EmitsRequestInstallForCursor(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.cursor = 1
	_, cmd := applyMsg(m, keyPress("i"))
	require.NotNil(t, cmd)
	got := cmd().(messages.RequestInstallMsg)
	require.Equal(t, []string{"Hack"}, got.Tags)
}

func TestInstallKey_WithSelection_EmitsBatch(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.selected["FiraCode"] = true
	m.selected["Hack"] = true
	_, cmd := applyMsg(m, keyPress("i"))
	got := cmd().(messages.RequestInstallMsg)
	require.ElementsMatch(t, []string{"FiraCode", "Hack"}, got.Tags)
}

func TestFilter_NarrowsVisible(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.filter = "fira"
	visible := m.Visible()
	require.Len(t, visible, 1)
	require.Equal(t, "FiraCode", visible[0].Name)
}

func TestSort_Cycles(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	require.Equal(t, SortByName, m.sort)
	m, _ = applyMsg(m, keyPress("s"))
	require.NotEqual(t, SortByName, m.sort)
}

func TestFontStateChanged_PatchesInPlace(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m, _ = applyMsg(m, messages.FontStateChangedMsg{
		Font: engine.FontInfo{Name: "Hack", Status: engine.StatusInstalled, Version: "v3.2.1"},
	})
	for _, f := range m.fonts {
		if f.Name == "Hack" {
			require.Equal(t, engine.StatusInstalled, f.Status)
			return
		}
	}
	t.Fatal("Hack not found")
}

func TestView_DoesNotPanicOnEmptyList(t *testing.T) {
	m := New(keys.Default())
	m.Width, m.Height = 40, 12
	_ = m.View()
}

func TestRender_ContainsAllNames(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.Width, m.Height = 40, 12
	s := stripANSI(m.View().Content)
	for _, name := range []string{"FiraCode", "Hack", "Iosevka", "JetBrainsMono"} {
		require.Contains(t, s, name)
	}
}

func applyMsg(m Model, msg tea.Msg) (Model, tea.Cmd) {
	out, cmd := m.Update(msg)
	return out.(Model), cmd
}

func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case " ":
		return tea.KeyPressMsg{Code: ' ', Text: " "}
	default:
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
}

func stripANSI(s string) string {
	return ansi.Strip(s)
}

// Avoid unused-import error if strings/ansi not consumed elsewhere.
var _ = strings.Contains
