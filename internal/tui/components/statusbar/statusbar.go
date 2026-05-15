// Package statusbar renders the bottom bar with key hints + activity badges.
//
// The component is stateless: callers set the public fields directly and call
// View(). Init and Update are provided only so the model satisfies tea.Model
// and can be embedded alongside other components.
package statusbar

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/theme"
)

// Model is the statusbar's tea.Model. Public fields are written by the app
// root model before each render; View() produces a single-line bar.
type Model struct {
	// Keys provides the bindings whose ShortHelp is rendered on the left.
	Keys keys.KeyMap

	// Width is the total cell width the bar should occupy. When zero or
	// negative, View falls back to a sensible default (80).
	Width int
	// InFlight is the number of in-progress engine operations to surface
	// as a badge on the right. Zero hides the badge.
	InFlight int
	// SelectionCount is the number of multi-selected items to surface as
	// a badge on the right. Zero hides the badge.
	SelectionCount int
}

// New constructs a statusbar with the given KeyMap and zeroed counters.
func New(k keys.KeyMap) Model { return Model{Keys: k} }

// Init is a no-op: the statusbar holds no state that needs initialisation.
func (m Model) Init() tea.Cmd { return nil }

// Update only reacts to WindowSizeMsg to track terminal width; everything else
// is ignored and the model is returned unchanged.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.Width = ws.Width
	}
	return m, nil
}

// View renders the statusbar as a single line of width m.Width with key hints
// on the left and "N ops" / "N selected" badges right-aligned.
func (m Model) View() tea.View {
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	badgeStyle := lipgloss.NewStyle().Foreground(theme.StatusInfo)

	parts := make([]string, 0, len(m.Keys.ShortHelp()))
	for _, b := range m.Keys.ShortHelp() {
		h := b.Help()
		if h.Key == "" {
			continue
		}
		parts = append(parts, hintStyle.Render(h.Key+" "+h.Desc))
	}
	hints := strings.Join(parts, "  ")

	badges := make([]string, 0, 2)
	if m.InFlight > 0 {
		badges = append(badges, badgeStyle.Render(fmt.Sprintf("%d ops", m.InFlight)))
	}
	if m.SelectionCount > 0 {
		badges = append(badges, badgeStyle.Render(fmt.Sprintf("%d selected", m.SelectionCount)))
	}
	right := strings.Join(badges, "  ")

	width := m.Width
	if width <= 0 {
		width = 80
	}
	bar := lipgloss.NewStyle().
		Width(width).
		Render(hints + spaceFill(width, hints, right) + right)
	return tea.NewView(bar)
}

// spaceFill returns the padding between left hints and right badges so that
// the badges land flush with the right edge. The pad is never less than 1 so
// the two halves never touch.
func spaceFill(width int, left, right string) string {
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return strings.Repeat(" ", pad)
}
