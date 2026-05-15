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

// Pre-rendered symbol strings cached at package init. lipgloss styles are
// immutable values so the rendered output is safe to share across goroutines.
var (
	SymOK   = lipgloss.NewStyle().Foreground(StatusOK).Render("✓")
	SymWarn = lipgloss.NewStyle().Foreground(StatusWarn).Render("⚠")
	SymFail = lipgloss.NewStyle().Foreground(StatusFail).Render("✗")
	SymSkip = lipgloss.NewStyle().Foreground(TextDim).Render("·")

	paneFocusedStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderFocus)
	paneDimStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(BorderDim)
)

// SymbolOK returns the cached green check mark for installed/healthy items.
func SymbolOK() string { return SymOK }

// SymbolWarn returns the cached amber warning glyph for stale/warn items.
func SymbolWarn() string { return SymWarn }

// SymbolFail returns the cached red failure glyph for fail/error items.
func SymbolFail() string { return SymFail }

// SymbolSkip returns the cached dim skip dot for inapplicable/N/A items.
func SymbolSkip() string { return SymSkip }

// PaneStyle returns a pre-built border style for the given focus state.
// The caller chains .Width/.Height/.Padding which clones the style
// (lipgloss v2 Style is a value type) so the cached styles remain
// unmodified across calls.
func PaneStyle(focused bool) lipgloss.Style {
	if focused {
		return paneFocusedStyle
	}
	return paneDimStyle
}

// SelectedRow highlights the cursor row in fontlist.
func SelectedRow() lipgloss.Style {
	return lipgloss.NewStyle().Background(SelectedBG).Foreground(TextHi)
}

// CheckedRow is the prefix on a multi-selected row.
func CheckedRow() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(StatusInfo)
}
