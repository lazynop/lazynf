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

func applyMsg(m Model, msg tea.Msg) (Model, tea.Cmd) {
	out, cmd := m.Update(msg)
	return out.(Model), cmd
}
