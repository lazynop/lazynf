package logpane

import (
	"errors"
	"os"
	"testing"

	"charm.land/bubbles/v2/spinner"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
)

func TestEngineEvent_Started_AppendsLine(t *testing.T) {
	m := New(nil)
	m.Width, m.Height = 80, 8
	out, _ := m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.StartedEvent{OpID: 1, Kind: "install", Target: "FiraCode"},
	})
	require.Contains(t, ansi.Strip(out.(Model).View().Content), "install")
}

func TestEngineEvent_Completed_AppendsAndClearsOp(t *testing.T) {
	m := New(nil)
	m.Width, m.Height = 80, 8
	out, _ := m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.ProgressEvent{Target: "X", Written: 1, Total: 2},
	})
	out, _ = out.(Model).Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.CompletedEvent{Target: "X", Kind: engine.CompletedSuccess, Detail: "ok"},
	})
	mm := out.(Model)
	require.NotContains(t, mm.ops, "X")
	require.Contains(t, ansi.Strip(mm.View().Content), "X")
}

func TestEngineEvent_Failed_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	file := NewFileLogger(dir)
	m := New(file)
	_, _ = m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.FailedEvent{Target: "Hack", Err: errors.New("boom")},
	})
	data, err := os.ReadFile(file.Path())
	require.NoError(t, err)
	require.Contains(t, string(data), "FAIL Hack: boom")
}

func TestInit_ReturnsNil(t *testing.T) {
	require.Nil(t, New(nil).Init())
}

func TestUpdate_UnknownMsg_NoOp(t *testing.T) {
	m := New(nil)
	out, cmd := m.Update("unknown")
	require.Nil(t, cmd)
	require.Equal(t, len(m.lines), len(out.(Model).lines))
}

func TestUpdate_SpinnerTick_ForwardsToOps(t *testing.T) {
	m := New(nil)
	m.ops["X"] = m.ensureOp("X")
	out, cmd := m.Update(spinner.TickMsg{ID: m.ops["X"].spinner.ID()})
	require.IsType(t, Model{}, out)
	require.NotNil(t, cmd)
}

func TestView_NotVisible_ReturnsEmpty(t *testing.T) {
	m := New(nil)
	m.Visible = false
	require.Equal(t, "", m.View().Content)
}

func TestView_ZeroDimensions_UsesDefaults(t *testing.T) {
	m := New(nil)
	require.NotEmpty(t, m.View().Content)
}

func TestView_WithOpSummary_ContainsPercent(t *testing.T) {
	m := New(nil)
	m.Width, m.Height = 80, 8
	out, _ := m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.ProgressEvent{Target: "JBM", Written: 1, Total: 4},
	})
	s := ansi.Strip(out.(Model).View().Content)
	require.Contains(t, s, "JBM")
	require.Contains(t, s, "25%")
}

func TestEngineEvent_Failed_NoFile_DoesNotPanic(t *testing.T) {
	m := New(nil)
	_, _ = m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.FailedEvent{Target: "Hack", Err: errors.New("boom")},
	})
}

func TestEngineEvent_Failed_NilErr_NoFileWrite(t *testing.T) {
	dir := t.TempDir()
	file := NewFileLogger(dir)
	m := New(file)
	_, _ = m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.FailedEvent{Target: "Hack", Err: nil},
	})
	data, err := os.ReadFile(file.Path())
	if err == nil {
		require.NotContains(t, string(data), "FAIL Hack")
	}
}

func TestEngineEvent_Canceled_AppendsAndClearsOp(t *testing.T) {
	m := New(nil)
	m.Width, m.Height = 80, 8
	out, _ := m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.ProgressEvent{Target: "Z", Written: 1, Total: 2},
	})
	out, _ = out.(Model).Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.CanceledEvent{Target: "Z"},
	})
	mm := out.(Model)
	require.NotContains(t, mm.ops, "Z")
	require.Contains(t, ansi.Strip(mm.View().Content), "canceled")
}

func TestEngineEvent_Log_AppendsLine(t *testing.T) {
	m := New(nil)
	m.Width, m.Height = 80, 8
	out, _ := m.Update(messages.EngineEventMsg{
		OpID: 1, Ev: engine.LogEvent{Target: "X", Level: engine.LevelInfo, Message: "hi"},
	})
	require.Contains(t, ansi.Strip(out.(Model).View().Content), "hi")
}

func TestTail_ShortSlice_ReturnsAll(t *testing.T) {
	require.Equal(t, []string{"a", "b"}, tail([]string{"a", "b"}, 5))
}

func TestTail_ZeroOrNegativeN_ReturnsAll(t *testing.T) {
	require.Equal(t, []string{"a", "b"}, tail([]string{"a", "b"}, 0))
	require.Equal(t, []string{"a", "b"}, tail([]string{"a", "b"}, -1))
}

func TestTail_LongSlice_ReturnsLastN(t *testing.T) {
	require.Equal(t, []string{"c", "d", "e"}, tail([]string{"a", "b", "c", "d", "e"}, 3))
}

func TestRingBuffer_DoesNotGrowUnbounded(t *testing.T) {
	m := New(nil)
	m.Width, m.Height = 80, 8
	for i := 0; i < 500; i++ {
		out, _ := m.Update(messages.EngineEventMsg{
			OpID: engine.OpID(i),
			Ev:   engine.LogEvent{Target: "X", Level: engine.LevelInfo, Message: "spam"},
		})
		m = out.(Model)
	}
	require.LessOrEqual(t, len(m.lines), ringSize)
}
