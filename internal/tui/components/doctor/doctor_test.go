package doctor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/stretchr/testify/require"
)

func TestSection_AccumulatesEvents(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{
		Section: "fc-cache", Status: engine.DoctorOK, Detail: "ok",
	})
	m, _ = applyMsg(m, messages.DoctorSectionMsg{
		Section: "catalog", Status: engine.DoctorFail, Detail: "stale",
		Action: engine.ActionRefreshCatalog,
	})
	require.Len(t, m.sections, 2)
}

func TestEnterOnActionable_EmitsRefreshCatalog(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{
		Section: "catalog", Status: engine.DoctorFail,
		Action: engine.ActionRefreshCatalog,
	})
	m.cursor = 0
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_, ok := cmd().(messages.RequestRefreshCatalogMsg)
	require.True(t, ok)
}

func TestEnterOnNonActionable_NoCmd(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{
		Section: "manifest", Status: engine.DoctorOK,
	})
	m.cursor = 0
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Nil(t, cmd)
}

func TestR_TriggersRerun(t *testing.T) {
	m := New(keys.Default())
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: 'r', Text: "r"})
	require.NotNil(t, cmd)
	_, ok := cmd().(messages.RequestDoctorMsg)
	require.True(t, ok)
}

func TestRender_ShowsSections(t *testing.T) {
	m := New(keys.Default())
	m.Width, m.Height = 80, 24
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "paths", Status: engine.DoctorOK, Detail: "OK"})
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "paths")
}

func TestInit_ReturnsNil(t *testing.T) {
	require.Nil(t, New(keys.Default()).Init())
}

func TestDown_MovesCursor(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "a", Status: engine.DoctorOK})
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "b", Status: engine.DoctorOK})
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	require.Equal(t, 1, m.cursor)
}

func TestDown_AtEnd_DoesNotMove(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "a", Status: engine.DoctorOK})
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	require.Equal(t, 0, m.cursor)
}

func TestUp_MovesCursor(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "a", Status: engine.DoctorOK})
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "b", Status: engine.DoctorOK})
	m.cursor = 1
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'k', Text: "k"})
	require.Equal(t, 0, m.cursor)
}

func TestUp_AtTop_DoesNotMove(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "a", Status: engine.DoctorOK})
	m, _ = applyMsg(m, tea.KeyPressMsg{Code: 'k', Text: "k"})
	require.Equal(t, 0, m.cursor)
}

func TestEnter_CursorOutOfRange_NoCmd(t *testing.T) {
	m := New(keys.Default())
	m.cursor = 5
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Nil(t, cmd)
}

func TestEnter_ActionRefreshFontCache_EmitsRequestDoctor(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{
		Section: "fc", Status: engine.DoctorFail, Action: engine.ActionRefreshFontCache,
	})
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_, ok := cmd().(messages.RequestDoctorMsg)
	require.True(t, ok)
}

func TestUnboundKey_NoCmd(t *testing.T) {
	m := New(keys.Default())
	_, cmd := applyMsg(m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	require.Nil(t, cmd)
}

func TestView_Empty_DoesNotPanic(t *testing.T) {
	m := New(keys.Default())
	require.NotEmpty(t, m.View().Content)
}

func TestView_Running_ShowsHint(t *testing.T) {
	m := New(keys.Default())
	m.Width, m.Height = 80, 24
	m.running = true
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "running")
}

func TestView_ZeroDimensions_UsesDefaults(t *testing.T) {
	m := New(keys.Default())
	require.NotEmpty(t, m.View().Content)
}

func TestR_ResetsSectionsAndCursor(t *testing.T) {
	m := New(keys.Default())
	m, _ = applyMsg(m, messages.DoctorSectionMsg{Section: "a", Status: engine.DoctorOK})
	m.cursor = 0
	m, cmd := applyMsg(m, tea.KeyPressMsg{Code: 'r', Text: "r"})
	require.NotNil(t, cmd)
	require.Empty(t, m.sections)
	require.Equal(t, 0, m.cursor)
	require.True(t, m.running)
}

func TestGlyphFor_AllStatuses(t *testing.T) {
	require.Equal(t, "✓", ansi.Strip(glyphFor(engine.DoctorOK)))
	require.Equal(t, "⚠", ansi.Strip(glyphFor(engine.DoctorWarn)))
	require.Equal(t, "✗", ansi.Strip(glyphFor(engine.DoctorFail)))
	require.Equal(t, "·", ansi.Strip(glyphFor(engine.DoctorSkip)))
}

func applyMsg(m Model, msg tea.Msg) (Model, tea.Cmd) {
	out, cmd := m.Update(msg)
	return out.(Model), cmd
}
