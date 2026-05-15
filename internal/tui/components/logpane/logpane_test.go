package logpane

import (
	"errors"
	"os"
	"testing"

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
