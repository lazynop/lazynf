// Package help renders a keybinding cheat-sheet overlay.
package help

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/theme"
)

// Package-level styles hoisted out of View to avoid per-frame allocation.
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.TextHi)
	keyStyle   = lipgloss.NewStyle().Foreground(theme.StatusInfo)
	dimStyle   = lipgloss.NewStyle().Foreground(theme.TextDim)
)

// Model renders the full-screen-centered help overlay.
type Model struct {
	// Keys provides the bindings rendered as the cheat-sheet table.
	Keys keys.KeyMap
	// Width is the total cell width to centre the overlay within. When zero
	// or negative, View falls back to a sensible default (80).
	Width int
	// Height is the total cell height to centre the overlay within. When
	// zero or negative, View falls back to a sensible default (24).
	Height int
}

// New constructs a help overlay reading from the given KeyMap.
func New(k keys.KeyMap) Model { return Model{Keys: k} }

// Init is a no-op (stateless).
func (m Model) Init() tea.Cmd { return nil }

// Update is a no-op (parent gates open/close via overlay state).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

// View renders the help table centered on the screen.
func (m Model) View() tea.View {
	rows := []string{titleStyle.Render("Key bindings"), ""}
	for _, row := range m.Keys.FullHelp() {
		var parts []string
		for _, b := range row {
			h := b.Help()
			parts = append(parts, keyStyle.Render(h.Key)+" "+dimStyle.Render(h.Desc))
		}
		rows = append(rows, strings.Join(parts, "   "))
	}
	rows = append(rows, "", dimStyle.Render("press ? to close"))

	body := strings.Join(rows, "\n")
	box := theme.PaneStyle(true).Padding(1, 2).Render(body)

	w, h := m.Width, m.Height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	centered := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
	return tea.NewView(centered)
}
