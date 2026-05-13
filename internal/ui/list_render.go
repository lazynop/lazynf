// Package ui — list rendering helpers for `lazynf list` and `lazynf list --installed`.
//
// Three modes are supported:
//   - RenderCatalogGrid: multi-column color-coded grid for TTY output.
//   - RenderInstalledTable: bordered table for TTY output.
//   - RenderCatalogPlain / RenderInstalledPlain: pipe-friendly one-per-line fallbacks.
package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/state"
)

const (
	// colPadding is the number of spaces appended after each cell.
	colPadding = 4

	// timestampLayout is the display format for InstalledAt timestamps.
	timestampLayout = "2006-01-02 15:04"
)

// isInstalled reports whether a FontInfo represents a font present on disk
// (any non-Available status counts as installed for rendering purposes).
func isInstalled(fi engine.FontInfo) bool {
	return fi.Status != engine.StatusAvailable
}

// releaseLabel returns the release string to display for a FontInfo:
// the ReleaseImported sentinel for imported fonts, otherwise the Version
// field as recorded in the manifest.
func releaseLabel(fi engine.FontInfo) string {
	if fi.Status == engine.StatusImported {
		return state.ReleaseImported
	}
	return fi.Version
}

// RenderCatalogGrid renders the full font catalog as a multi-column
// color-coded grid that fits termWidth terminal columns.
//
// infos must be sorted (alphabetical) — engine.List already returns sorted
// slices.
//
// Each cell is just the font name, colored by install status:
//   - green/bold if installed with a real release tag
//   - yellow/bold if imported (ReleaseImported sentinel)
//   - plain if not installed (StatusAvailable)
//
// A one-line legend is appended after the grid.
func RenderCatalogGrid(infos []engine.FontInfo, termWidth int) string {
	if len(infos) == 0 {
		return ""
	}

	// Compute the width of the widest font name.
	nameWidth := 0
	for _, fi := range infos {
		if w := utf8.RuneCountInString(fi.Name); w > nameWidth {
			nameWidth = w
		}
	}

	colWidth := nameWidth + colPadding
	numCols := max(termWidth/colWidth, 1)

	numFonts := len(infos)
	numRows := (numFonts + numCols - 1) / numCols

	// Build each column as a slice of strings (one per row).
	columns := make([]string, numCols)
	for col := range numCols {
		var sb strings.Builder
		for row := range numRows {
			// Column-major order: top-to-bottom then left-to-right.
			idx := col*numRows + row
			if idx >= numFonts {
				// Pad empty cells so columns stay aligned.
				sb.WriteString(strings.Repeat(" ", colWidth))
			} else {
				sb.WriteString(renderCell(infos[idx], colWidth))
			}
			if row < numRows-1 {
				sb.WriteRune('\n')
			}
		}
		columns[col] = sb.String()
	}

	grid := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	// Legend line.
	legend := StyleDim.Render("Legend: ") +
		StyleSuccess.Render("green") +
		StyleDim.Render(" = installed   ") +
		StyleWarn.Render("yellow") +
		StyleDim.Render(" = imported   (use 'lazynf list --installed' for details)")

	return grid + "\n\n" + legend
}

// renderCell returns a single grid cell of exactly colWidth visible characters.
// The cell is just the font name, colored by install status (no inline status text).
func renderCell(fi engine.FontInfo, colWidth int) string {
	var sb strings.Builder

	// Font name — colored by install status.
	switch {
	case fi.Status == engine.StatusImported:
		sb.WriteString(StyleWarn.Render(fi.Name))
	case isInstalled(fi):
		sb.WriteString(StyleSuccess.Render(fi.Name))
	default:
		sb.WriteString(fi.Name)
	}

	// Pad to colWidth (name occupies nameWidth visible chars).
	remaining := colWidth - utf8.RuneCountInString(fi.Name)
	if remaining > 0 {
		sb.WriteString(strings.Repeat(" ", remaining))
	}

	return sb.String()
}

// RenderInstalledTable renders the installed-fonts list as a bordered table
// with columns: FONT | RELEASE | INSTALLED AT.
//
// infos must contain only entries the caller wants displayed (typically the
// engine.List output filtered to non-Available statuses). It must be sorted.
func RenderInstalledTable(infos []engine.FontInfo) string {
	if len(infos) == 0 {
		return "(no fonts installed)"
	}

	// cellStyle is the base for every cell: 1 char of horizontal padding so
	// content doesn't touch the borders.
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return cellStyle.Bold(true)
			}
			if col == 1 {
				// Release column: color by value.
				fi := infos[row]
				if fi.Status == engine.StatusImported {
					return cellStyle.Inherit(StyleWarn)
				}
				return cellStyle.Inherit(StyleSuccess)
			}
			return cellStyle
		}).
		Headers("FONT", "RELEASE", "INSTALLED AT")

	for _, fi := range infos {
		ts := fi.InstalledAt.Local().Format(timestampLayout)
		t.Row(fi.Name, releaseLabel(fi), ts)
	}

	return t.Render()
}

// RenderCatalogPlain renders the font catalog as plain text: one font name per
// line, no ANSI sequences. Suitable for piped / non-TTY output.
func RenderCatalogPlain(infos []engine.FontInfo) string {
	if len(infos) == 0 {
		return ""
	}
	names := make([]string, len(infos))
	for i, fi := range infos {
		names[i] = fi.Name
	}
	return strings.Join(names, "\n")
}

// RenderInstalledPlain renders the installed-font list as tab-separated
// `<name>\t<release>` lines with no ANSI sequences. Suitable for awk/cut.
//
// infos must contain only the rows the caller wants emitted (typically
// engine.List output filtered to non-Available statuses).
func RenderInstalledPlain(infos []engine.FontInfo) string {
	if len(infos) == 0 {
		return ""
	}
	lines := make([]string, len(infos))
	for i, fi := range infos {
		lines[i] = fmt.Sprintf("%s\t%s", fi.Name, releaseLabel(fi))
	}
	return strings.Join(lines, "\n")
}
