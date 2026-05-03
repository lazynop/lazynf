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
	// statusFieldWidth is the fixed width reserved for the status indicator
	// (e.g. "✓v3.4.0" or "✓imported"). 12 chars covers the longest case.
	statusFieldWidth = 12

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

	colWidth := nameWidth + statusFieldWidth + colPadding
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

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

// renderCell returns a single grid cell of exactly colWidth visible characters.
func renderCell(name string, entry state.InstalledFont, isInstalled bool, nameWidth, colWidth int) string {
	var sb strings.Builder

	// Font name part.
	if isInstalled {
		sb.WriteString(StyleBold.Render(name))
	} else {
		sb.WriteString(name)
	}

	// Pad name to nameWidth.
	namePad := nameWidth - utf8.RuneCountInString(name)
	if namePad > 0 {
		sb.WriteString(strings.Repeat(" ", namePad))
	}

	// Status indicator.
	statusText := ""
	if isInstalled {
		if entry.Release == state.ReleaseImported {
			indicator := "✓imported"
			statusText = StyleWarn.Render(indicator)
		} else {
			indicator := "✓" + entry.Release
			statusText = StyleSuccess.Render(indicator)
		}
	}

	sb.WriteString(statusText)

	// Compute the visible width consumed so far (name + status), then pad to colWidth.
	visibleStatusWidth := 0
	if isInstalled {
		if entry.Release == state.ReleaseImported {
			visibleStatusWidth = utf8.RuneCountInString("✓imported")
		} else {
			visibleStatusWidth = utf8.RuneCountInString("✓" + entry.Release)
		}
	}
	used := nameWidth + visibleStatusWidth
	remaining := colWidth - used
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
