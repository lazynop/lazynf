// Package doctor renders the diagnostic pane (DoctorSectionMsg accumulator)
// with actionable enter-on-section semantics.
package doctor

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/lazynop/lazynf/internal/tui/theme"
)

// Section is one accumulated check result the pane displays.
type Section struct {
	Name   string
	Title  string
	Status engine.DoctorStatus
	Detail string
	Hint   string
	Action engine.DoctorAction
}

// Model owns the cumulated sections + cursor.
type Model struct {
	Keys     keys.KeyMap
	sections []Section
	cursor   int
	running  bool
	Width    int
	Height   int
	Focused  bool
}

// New constructs an empty doctor pane.
func New(k keys.KeyMap) Model { return Model{Keys: k} }

// Init is a no-op (sections arrive via DoctorSectionMsg).
func (m Model) Init() tea.Cmd { return nil }

// Update routes incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case messages.DoctorSectionMsg:
		m.running = false
		m.sections = append(m.sections, Section{
			Name:   x.Section,
			Title:  x.Title,
			Status: x.Status,
			Detail: x.Detail,
			Hint:   x.Hint,
			Action: x.Action,
		})
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(x)
	}
	return m, nil
}

// handleKey processes keyboard input for navigation and actions.
func (m Model) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, m.Keys.Down):
		if m.cursor < len(m.sections)-1 {
			m.cursor++
		}
		return m, nil
	case key.Matches(k, m.Keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case k.Code == tea.KeyEnter:
		if m.cursor >= len(m.sections) {
			return m, nil
		}
		switch m.sections[m.cursor].Action {
		case engine.ActionRefreshCatalog:
			return m, messages.Cmd(messages.RequestRefreshCatalogMsg{})
		case engine.ActionRefreshFontCache:
			return m, messages.Cmd(messages.RequestDoctorMsg{})
		}
		return m, nil
	case k.Code == 'r':
		m.sections = nil
		m.cursor = 0
		m.running = true
		return m, messages.Cmd(messages.RequestDoctorMsg{})
	}
	return m, nil
}

// View renders the pane centered on screen.
func (m Model) View() tea.View {
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.TextHi).Render("Doctor")
	dim := lipgloss.NewStyle().Foreground(theme.TextDim)

	rows := []string{title, ""}
	for i, s := range m.sections {
		glyph := glyphFor(s.Status)
		line := glyph + " " + s.Name
		if s.Detail != "" {
			line += "  " + dim.Render(s.Detail)
		}
		if s.Action != engine.ActionNone {
			line += "  " + dim.Render("(enter to fix)")
		}
		if i == m.cursor {
			line = theme.SelectedRow().Render(line)
		}
		rows = append(rows, line)
	}
	if m.running {
		rows = append(rows, "", dim.Render("running..."))
	}
	rows = append(rows, "", dim.Render("r: re-run   q: back"))

	body := strings.Join(rows, "\n")
	box := theme.PaneStyle(true).Padding(1, 2).Render(body)

	w, h := m.Width, m.Height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return tea.NewView(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box))
}

// glyphFor returns the themed status glyph for a DoctorStatus value.
func glyphFor(s engine.DoctorStatus) string {
	switch s {
	case engine.DoctorOK:
		return theme.SymbolOK()
	case engine.DoctorWarn:
		return theme.SymbolWarn()
	case engine.DoctorFail:
		return theme.SymbolFail()
	}
	return theme.SymbolSkip()
}
