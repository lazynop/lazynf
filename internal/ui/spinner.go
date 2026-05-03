package ui

import (
	"sync"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// Spinner runs a bubbletea spinner in its own goroutine. Single-use.
// Use NewSpinner, then call Start, then Stop when the operation completes.
type Spinner struct {
	label  string
	prog   *tea.Program
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

type spinnerStop struct{ ok bool }

type spinnerModel struct {
	s        spinner.Model
	label    string
	finished bool
	ok       bool
}

func newSpinnerModel(label string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return spinnerModel{s: s, label: label}
}

func (m spinnerModel) Init() tea.Cmd { return m.s.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerStop:
		m.finished = true
		m.ok = msg.ok
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.s, cmd = m.s.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() tea.View {
	if m.finished {
		icon := StyleSuccess.Render("✓")
		if !m.ok {
			icon = StyleFailure.Render("✗")
		}
		return tea.NewView(icon + " " + m.label)
	}
	return tea.NewView(m.s.View() + " " + m.label)
}

// NewSpinner returns an unstarted spinner with the given label.
func NewSpinner(label string) *Spinner {
	return &Spinner{label: label, done: make(chan struct{})}
}

// Start launches the spinner in a goroutine. Non-blocking.
// Calling Start more than once is a no-op.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.prog != nil {
		s.mu.Unlock()
		return // already started; single-use
	}
	s.prog = tea.NewProgram(newSpinnerModel(s.label))
	s.mu.Unlock()
	go func() {
		_, _ = s.prog.Run()
		// Mark closed so subsequent Stop calls are no-ops even if the program
		// exited via Ctrl-C (kill path) rather than via spinnerStop.
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.done)
	}()
}

// Stop ends the spinner with the given outcome and waits for the goroutine to exit.
func (s *Spinner) Stop(ok bool) {
	s.mu.Lock()
	if s.closed || s.prog == nil {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.prog.Send(spinnerStop{ok: ok})
	s.mu.Unlock()
	<-s.done
}
