// Package ui — list rendering helpers for `vellum list` and `vellum list --installed`.
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
	"github.com/lazynop/vellum/internal/state"
)

// InstalledSet is a lookup map from font name to its manifest entry.
type InstalledSet = map[string]state.InstalledFont

const (
	// colPadding is the number of spaces appended after each cell.
	colPadding = 4

	// timestampLayout is the display format for InstalledAt timestamps.
	timestampLayout = "2006-01-02 15:04"
)

// RenderCatalogGrid renders the full font catalog as a multi-column
// color-coded grid that fits termWidth terminal columns.
//
// fonts must be sorted (alphabetical). installed maps font names present in the
// manifest; a nil map is treated as empty (no installed fonts).
//
// Each cell is just the font name, colored by install status:
//   - green/bold if installed with a real release tag
//   - yellow/bold if installed with the ReleaseImported sentinel
//   - plain if not installed
//
// A one-line legend is appended after the grid.
func RenderCatalogGrid(fonts []string, installed InstalledSet, termWidth int) string {
	if len(fonts) == 0 {
		return ""
	}

	// Compute the width of the widest font name.
	nameWidth := 0
	for _, n := range fonts {
		if w := utf8.RuneCountInString(n); w > nameWidth {
			nameWidth = w
		}
	}

	colWidth := nameWidth + colPadding
	numCols := max(termWidth/colWidth, 1)

	numFonts := len(fonts)
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
				name := fonts[idx]
				entry, isInstalled := installed[name]
				sb.WriteString(renderCell(name, entry, isInstalled, nameWidth, colWidth))
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
		StyleDim.Render(" = imported   (use 'vellum list --installed' for details)")

	return grid + "\n\n" + legend
}

// renderCell returns a single grid cell of exactly colWidth visible characters.
// The cell is just the font name, colored by install status (no inline status text).
func renderCell(name string, entry state.InstalledFont, isInstalled bool, nameWidth, colWidth int) string {
	var sb strings.Builder

	// Font name — colored by install status.
	if isInstalled {
		if entry.Release == state.ReleaseImported {
			sb.WriteString(StyleWarn.Render(name))
		} else {
			sb.WriteString(StyleSuccess.Render(name))
		}
	} else {
		sb.WriteString(name)
	}

	// Pad to colWidth (name occupies nameWidth visible chars).
	remaining := colWidth - utf8.RuneCountInString(name)
	if remaining > 0 {
		sb.WriteString(strings.Repeat(" ", remaining))
	}

	return sb.String()
}

// RenderInstalledTable renders the installed-fonts manifest as a bordered
// table with columns: FONT | RELEASE | INSTALLED AT.
//
// fonts must be sorted. installed is the full manifest map.
func RenderInstalledTable(fonts []string, installed InstalledSet) string {
	if len(fonts) == 0 {
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
				name := fonts[row]
				if entry, ok := installed[name]; ok {
					if entry.Release == state.ReleaseImported {
						return cellStyle.Inherit(StyleWarn)
					}
					return cellStyle.Inherit(StyleSuccess)
				}
			}
			return cellStyle
		}).
		Headers("FONT", "RELEASE", "INSTALLED AT")

	for _, name := range fonts {
		entry := installed[name]
		ts := entry.InstalledAt.Local().Format(timestampLayout)
		t.Row(name, entry.Release, ts)
	}

	return t.Render()
}

// RenderCatalogPlain renders the font catalog as plain text: one font name per
// line, no ANSI sequences. Suitable for piped / non-TTY output.
func RenderCatalogPlain(fonts []string) string {
	if len(fonts) == 0 {
		return ""
	}
	return strings.Join(fonts, "\n")
}

// RenderInstalledPlain renders the installed-font list as tab-separated
// `<name>\t<release>` lines with no ANSI sequences. Suitable for awk/cut.
func RenderInstalledPlain(fonts []string, installed InstalledSet) string {
	if len(fonts) == 0 {
		return ""
	}
	lines := make([]string, len(fonts))
	for i, name := range fonts {
		entry := installed[name]
		lines[i] = fmt.Sprintf("%s\t%s", name, entry.Release)
	}
	return strings.Join(lines, "\n")
}
