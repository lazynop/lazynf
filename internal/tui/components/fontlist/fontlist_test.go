package fontlist

import (
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

func TestInit_ReturnsNil(t *testing.T) {
	require.Nil(t, New(keys.Default()).Init())
}

func TestUp_AtTop_DoesNotMove(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m, _ = applyMsg(m, keyPress("k"))
	require.Equal(t, 0, m.cursor)
}

func TestUp_AfterDown_DecrementsCursor(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.cursor = 2
	m, _ = applyMsg(m, keyPress("k"))
	require.Equal(t, 1, m.cursor)
}

func TestDown_AtBottom_DoesNotMove(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.cursor = 3
	m, _ = applyMsg(m, keyPress("j"))
	require.Equal(t, 3, m.cursor)
}

func TestTop_MovesCursorToZero(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.cursor = 3
	m, _ = applyMsg(m, keyPress("g"))
	require.Equal(t, 0, m.cursor)
}

func TestBottom_MovesCursorToLast(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'G', Text: "G"})
	require.Equal(t, 3, m.cursor)
}

func TestBottom_EmptyList_StaysAtZero(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'G', Text: "G"})
	require.Equal(t, 0, m.cursor)
}

func TestFilter_EntersFilterEditing(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m, _ = applyMsg(m, keyPress("/"))
	require.True(t, m.FilterEditing)
}

func TestSortCycle_WrapsAround(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.sort = SortBySize
	m, _ = applyMsg(m, keyPress("s"))
	require.Equal(t, SortByName, m.sort)
}

func TestVisible_SortByStatus_Orders(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.sort = SortByStatus
	v := m.Visible()
	require.Equal(t, engine.StatusAvailable, v[0].Status)
}

func TestVisible_SortBySize_Orders(t *testing.T) {
	fonts := []engine.FontInfo{
		{Name: "Small", Size: 100},
		{Name: "Big", Size: 1000},
	}
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: fonts})
	m.sort = SortBySize
	v := m.Visible()
	require.Equal(t, "Big", v[0].Name)
}

func TestClearSelect_ClearsSelectionMap(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.selected["FiraCode"] = true
	m, cmd := applyMsg(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.Empty(t, m.selected)
	require.Equal(t, 0, cmd().(messages.SelectionChangedMsg).Count)
}

func TestUpdateKey_EmitsRequestUpdate(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.cursor = 0
	_, cmd := applyMsg(m, keyPress("u"))
	require.NotNil(t, cmd)
	got := cmd().(messages.RequestUpdateMsg)
	require.Equal(t, []string{"FiraCode"}, got.Tags)
}

func TestRemoveKey_EmitsRequestRemove(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	_, cmd := applyMsg(m, keyPress("r"))
	require.NotNil(t, cmd)
	got := cmd().(messages.RequestRemoveMsg)
	require.False(t, got.Purge)
}

func TestPurgeKey_EmitsRequestRemoveWithPurge(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: 'P', Text: "P"})
	require.NotNil(t, cmd)
	got := cmd().(messages.RequestRemoveMsg)
	require.True(t, got.Purge)
}

func TestImportKey_EmitsRequestImport(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: 'I', Text: "I"})
	require.NotNil(t, cmd)
	got := cmd().(messages.RequestImportMsg)
	require.True(t, got.Detect)
}

func TestHandleKey_UnboundKey_NoCmd(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	_, cmd := applyMsg(m, keyPress("z"))
	require.Nil(t, cmd)
}

func TestFilterMode_BackspaceShrinksFilter(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.FilterEditing = true
	m.filter = "fira"
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	require.Equal(t, "fir", m.filter)
}

func TestFilterMode_BackspaceEmpty_NoChange(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.FilterEditing = true
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	require.Equal(t, "", m.filter)
}

func TestFilterMode_PrintableAppends(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.FilterEditing = true
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'f', Text: "f"})
	require.Equal(t, "f", m.filter)
}

func TestFilterMode_NonPrintableIgnored(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.FilterEditing = true
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'é'})
	require.Equal(t, "", m.filter)
}

func TestFilterMode_EnterExitsFilter(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.FilterEditing = true
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.False(t, m.FilterEditing)
}

func TestFilterMode_EscapeClearsFilterAndExits(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.FilterEditing = true
	m.filter = "fira"
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	require.False(t, m.FilterEditing)
	require.Equal(t, "", m.filter)
}

func TestFontStateChanged_MissingFont_NoChange(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m, _ = applyMsg(m, messages.FontStateChangedMsg{
		Font: engine.FontInfo{Name: "DoesNotExist", Status: engine.StatusInstalled},
	})
	require.Len(t, m.fonts, 4)
}

func TestView_FilterEditingShowsBuffer(t *testing.T) {
	m := New(keys.Default())
	m.Width, m.Height = 40, 12
	m, _ = applyMsg(m, messages.FontsLoadedMsg{Fonts: makeFonts()})
	m.FilterEditing = true
	m.filter = "fi"
	s := stripANSI(m.View().Content)
	require.Contains(t, s, "/fi")
}

func TestWindowed_ViewportZero_ReturnsAll(t *testing.T) {
	s, e := windowed(0, 0, 5)
	require.Equal(t, 0, s)
	require.Equal(t, 5, e)
}

func TestWindowed_ViewportNegative_ReturnsAll(t *testing.T) {
	s, e := windowed(0, -1, 5)
	require.Equal(t, 0, s)
	require.Equal(t, 5, e)
}

func TestWindowed_ViewportLargerThanTotal_ReturnsAll(t *testing.T) {
	s, e := windowed(0, 10, 5)
	require.Equal(t, 0, s)
	require.Equal(t, 5, e)
}

func TestWindowed_CursorWithinFirstWindow(t *testing.T) {
	s, e := windowed(2, 5, 20)
	require.Equal(t, 0, s)
	require.Equal(t, 5, e)
}

func TestWindowed_CursorScrollsWindow(t *testing.T) {
	s, e := windowed(7, 3, 20)
	require.Equal(t, 5, s)
	require.Equal(t, 8, e)
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
