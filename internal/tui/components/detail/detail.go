// Package detail renders the right-hand FontInfo detail pane.
package detail

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/lazynop/lazynf/internal/tui/theme"
)

// Model renders the right-hand pane that displays the highlighted font.
type Model struct {
	Current       *engine.FontInfo
	Width, Height int
	Focused       bool
}

// New constructs an empty detail pane (Current is nil until a Highlight arrives).
func New() Model { return Model{} }

// Init is a no-op (the pane is purely reactive).
func (m Model) Init() tea.Cmd { return nil }

// Update reacts to FontHighlightedMsg by replacing Current.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case messages.FontHighlightedMsg:
		m.Current = x.Font
	}
	return m, nil
}

// View renders the pane.
func (m Model) View() tea.View {
	dim := lipgloss.NewStyle().Foreground(theme.TextDim)
	hi := lipgloss.NewStyle().Foreground(theme.TextHi).Bold(true)
	label := lipgloss.NewStyle().Foreground(theme.TextDim).Width(10)

	var body string
	if m.Current == nil {
		body = dim.Render("no font selected")
	} else {
		fi := m.Current
		rows := []string{
			hi.Render(fi.Name),
			"",
			label.Render("status:") + statusText(fi.Status),
		}
		if fi.Version != "" {
			rows = append(rows, label.Render("version:")+fi.Version)
		}
		if fi.LatestVersion != "" && fi.LatestVersion != fi.Version {
			rows = append(rows, label.Render("latest:")+fi.LatestVersion)
		}
		if len(fi.Files) > 0 {
			rows = append(rows, label.Render("files:")+fmt.Sprintf("%d  (%s)", len(fi.Files), humanSize(fi.Size)))
		}
		if !fi.InstalledAt.IsZero() {
			rows = append(rows, label.Render("since:")+fi.InstalledAt.Format("2006-01-02"))
		}
		if fi.Dir != "" {
			rows = append(rows, label.Render("path:")+fi.Dir)
		}
		body = strings.Join(rows, "\n")
	}

	border := theme.PaneStyle(m.Focused).
		Width(m.Width).
		Height(m.Height).
		Padding(1, 2)
	return tea.NewView(border.Render(body))
}

// statusText returns a labelled string for a font status (symbol + word).
func statusText(s engine.FontStatus) string {
	switch s {
	case engine.StatusInstalled:
		return theme.SymbolOK() + " installed"
	case engine.StatusStale:
		return theme.SymbolWarn() + " update available"
	case engine.StatusImported:
		return "○ imported"
	case engine.StatusAvailable:
		return "  available"
	case engine.StatusUnknown:
		return theme.SymbolFail() + " unknown (not in catalog)"
	}
	return ""
}

// humanSize formats a byte count as a binary-prefix human-readable string.
func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
