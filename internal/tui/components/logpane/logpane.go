package logpane

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/lazynop/lazynf/internal/tui/theme"
)

const ringSize = 200

// opState tracks per-target progress for the currently-streaming engine ops.
type opState struct {
	target  string
	spinner spinner.Model
	bar     progress.Model
	pct     float64
}

// Model renders the bottom log pane: ring buffer of recent lines, per-op
// spinners + progress bars, and an optional on-disk failure log.
type Model struct {
	lines []string
	ops   map[string]opState
	file  *FileLogger

	// Width is the cell width allotted to the pane.
	Width int
	// Height is the cell height allotted to the pane.
	Height int
	// Visible toggles rendering: when false, View returns an empty view.
	Visible bool
	// Focused lights the pane border when the pane has keyboard focus.
	Focused bool
}

// New constructs the logpane. file may be nil to disable on-disk persistence.
func New(file *FileLogger) Model {
	return Model{
		lines:   make([]string, 0, ringSize),
		ops:     map[string]opState{},
		file:    file,
		Visible: true,
	}
}

// Init returns nil; ticks start once the first op spins up.
func (m Model) Init() tea.Cmd { return nil }

// Update routes engine events and spinner ticks.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case messages.EngineEventMsg:
		return m.handleEngineEvent(x), nil
	case spinner.TickMsg:
		cmds := make([]tea.Cmd, 0, len(m.ops))
		for t, op := range m.ops {
			var c tea.Cmd
			op.spinner, c = op.spinner.Update(x)
			m.ops[t] = op
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// handleEngineEvent updates the ring buffer and op map based on an engine event.
func (m Model) handleEngineEvent(msg messages.EngineEventMsg) Model {
	switch ev := msg.Ev.(type) {
	case engine.StartedEvent:
		m.append(fmt.Sprintf("-> %s %s", ev.Kind, ev.Target))
	case engine.LogEvent:
		m.append(fmt.Sprintf("  %s %s", ev.Target, ev.Message))
	case engine.ProgressEvent:
		op := m.ensureOp(ev.Target)
		if ev.Total > 0 {
			op.pct = float64(ev.Written) / float64(ev.Total)
		}
		m.ops[ev.Target] = op
	case engine.CompletedEvent:
		m.append(theme.SymbolOK() + " " + ev.Target + " " + ev.Detail)
		delete(m.ops, ev.Target)
	case engine.FailedEvent:
		line := theme.SymbolFail() + " " + ev.Target
		if ev.Err != nil {
			line += ": " + ev.Err.Error()
		}
		m.append(line)
		delete(m.ops, ev.Target)
		if m.file != nil && ev.Err != nil {
			_ = m.file.Write(fmt.Sprintf("FAIL %s: %s", ev.Target, ev.Err.Error()))
		}
	case engine.CanceledEvent:
		m.append(theme.SymbolSkip() + " " + ev.Target + " canceled")
		delete(m.ops, ev.Target)
	}
	return m
}

// ensureOp returns the existing opState for target, or creates a fresh one
// with a Dot spinner and a default-blend progress bar.
func (m *Model) ensureOp(target string) opState {
	if op, ok := m.ops[target]; ok {
		return op
	}
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	bar := progress.New(progress.WithDefaultBlend())
	return opState{target: target, spinner: sp, bar: bar}
}

// append pushes a single line onto the ring, evicting the oldest entry when
// the buffer is full.
func (m *Model) append(line string) {
	stamp := time.Now().Format("15:04:05")
	full := lipgloss.NewStyle().Foreground(theme.TextDim).Render(stamp+" ") + line
	m.lines = append(m.lines, full)
	if len(m.lines) > ringSize {
		m.lines = m.lines[len(m.lines)-ringSize:]
	}
}

// View renders the pane: an optional summary of in-flight ops on the first
// line followed by the tail of the ring buffer.
func (m Model) View() tea.View {
	if !m.Visible {
		return tea.NewView("")
	}
	w, h := m.Width, m.Height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 8
	}
	body := tail(m.lines, h-2)
	rendered := strings.Join(body, "\n")
	if len(m.ops) > 0 {
		summary := make([]string, 0, len(m.ops))
		for _, op := range m.ops {
			summary = append(summary, fmt.Sprintf("%s %s %d%%", op.spinner.View(), op.target, int(op.pct*100)))
		}
		rendered = strings.Join(summary, "  ") + "\n" + rendered
	}
	border := theme.PaneStyle(m.Focused).Width(w).Height(h).Padding(0, 1)
	return tea.NewView(border.Render(rendered))
}

// tail returns the last n entries of s (or all of s if shorter).
func tail(s []string, n int) []string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
