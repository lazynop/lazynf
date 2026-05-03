package ui

import (
	"fmt"
	"sync"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
)

// ProgressTracker drives a bubbletea-based progress bar in its own goroutine.
// Use NewProgress, then call Start, then Update from any goroutine, then Finish or Fail.
//
// The tracker is single-use: after Finish/Fail, create a new one.
type ProgressTracker struct {
	label  string
	prog   *tea.Program
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

// progressMsg is a percentage update (0.0 .. 1.0) plus a label suffix.
type progressMsg struct {
	pct  float64
	note string
}

type doneMsg struct {
	ok     bool
	reason string
}

type progressModel struct {
	bar      progress.Model
	pct      float64
	label    string
	note     string
	finished bool
	ok       bool
	reason   string
}

func newProgressModel(label string) progressModel {
	return progressModel{
		bar:   progress.New(progress.WithDefaultBlend()),
		label: label,
	}
}

func (m progressModel) Init() tea.Cmd { return nil }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		m.pct = msg.pct
		m.note = msg.note
		return m, nil
	case doneMsg:
		m.finished = true
		m.ok = msg.ok
		m.reason = msg.reason
		return m, tea.Quit
	case tea.KeyMsg:
		// Allow Ctrl-C to abort the visual program (the underlying ctx is the user's signal).
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m progressModel) View() tea.View {
	icon := "⏳"
	if m.finished {
		if m.ok {
			icon = StyleSuccess.Render("✓")
		} else {
			icon = StyleFailure.Render("✗")
		}
	}
	suffix := ""
	if m.note != "" {
		suffix = " " + StyleDim.Render(m.note)
	}
	if m.finished && !m.ok && m.reason != "" {
		suffix = " — " + m.reason
	}
	return tea.NewView(fmt.Sprintf("%s %s %s%s", icon, m.label, m.bar.ViewAs(m.pct), suffix))
}

// NewProgress returns an unstarted tracker for a single labelled bar.
func NewProgress(label string) *ProgressTracker {
	return &ProgressTracker{
		label: label,
		done:  make(chan struct{}),
	}
}

// Start launches the bubbletea program in a goroutine. Non-blocking.
// Calling Start more than once is a no-op.
func (p *ProgressTracker) Start() {
	p.mu.Lock()
	if p.prog != nil {
		p.mu.Unlock()
		return // already started; single-use
	}
	p.prog = tea.NewProgram(newProgressModel(p.label))
	p.mu.Unlock()
	go func() {
		_, _ = p.prog.Run()
		// Mark closed so subsequent Update/Finish/Fail are no-ops even if
		// the program exited via Ctrl-C (kill path) rather than via doneMsg.
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		close(p.done)
	}()
}

// Update reports a fractional progress value (0..1). note is optional suffix text.
// Safe to call concurrently with the program loop.
func (p *ProgressTracker) Update(pct float64, note string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.prog == nil {
		return
	}
	p.prog.Send(progressMsg{pct: pct, note: note})
}

// Finish marks the bar successful and waits for the bubbletea goroutine to exit.
func (p *ProgressTracker) Finish() {
	p.close(true, "")
}

// Fail marks the bar failed (with a short reason) and waits for exit.
func (p *ProgressTracker) Fail(reason string) {
	p.close(false, reason)
}

func (p *ProgressTracker) close(ok bool, reason string) {
	p.mu.Lock()
	if p.closed || p.prog == nil {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.prog.Send(doneMsg{ok: ok, reason: reason})
	p.mu.Unlock()
	<-p.done
}
