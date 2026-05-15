// Package theme exposes the lipgloss styles used across TUI components.
// Lipgloss v2 styles are plain values; downsampling happens at output.
package theme

import "charm.land/lipgloss/v2"

// Palette colours used by TUI components. Border colours light up the focused
// pane; status colours are shared between doctor and fontlist row prefixes.
var (
	// BorderFocus is the border colour for the focused pane.
	BorderFocus = lipgloss.Color("#7c3aed") // violet
	// BorderDim is the border colour for unfocused panes.
	BorderDim = lipgloss.Color("#374151") // slate
	// TextDim is used for secondary/help text.
	TextDim = lipgloss.Color("#9ca3af")
	// TextHi is used for primary/emphasised text.
	TextHi = lipgloss.Color("#f3f4f6")
	// StatusOK indicates a successful state (green).
	StatusOK = lipgloss.Color("#22c55e")
	// StatusWarn indicates a warning state (amber).
	StatusWarn = lipgloss.Color("#f59e0b")
	// StatusFail indicates a failure state (red).
	StatusFail = lipgloss.Color("#ef4444")
	// StatusInfo indicates an informational state (blue).
	StatusInfo = lipgloss.Color("#60a5fa")
	// Cursor is the colour of the active cursor row marker.
	Cursor = lipgloss.Color("#7c3aed")
	// SelectedBG is the background colour of the cursor row.
	SelectedBG = lipgloss.Color("#1e293b") // slate-800
)

// PaneStyle returns the border style for a pane, lit when focused.
func PaneStyle(focused bool) lipgloss.Style {
	color := BorderDim
	if focused {
		color = BorderFocus
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color)
}

// SymbolOK returns the green check used to mark a successful state.
func SymbolOK() string { return lipgloss.NewStyle().Foreground(StatusOK).Render("✓") }

// SymbolWarn returns the amber warning glyph.
func SymbolWarn() string { return lipgloss.NewStyle().Foreground(StatusWarn).Render("⚠") }

// SymbolFail returns the red failure glyph.
func SymbolFail() string { return lipgloss.NewStyle().Foreground(StatusFail).Render("✗") }

// SymbolSkip returns the dim dot used to mark a skipped/no-op state.
func SymbolSkip() string { return lipgloss.NewStyle().Foreground(TextDim).Render("·") }

// SelectedRow highlights the cursor row in fontlist.
func SelectedRow() lipgloss.Style {
	return lipgloss.NewStyle().Background(SelectedBG).Foreground(TextHi)
}

// CheckedRow is the prefix on a multi-selected row.
func CheckedRow() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(StatusInfo)
}
