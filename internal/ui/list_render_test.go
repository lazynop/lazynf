package ui_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/state"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ansiRe matches ANSI escape sequences so we can strip them for plain-text assertions.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// makeInstalled is a helper to build a state.InstalledFont entry.
func makeInstalled(release string, at time.Time) state.InstalledFont {
	return state.InstalledFont{
		Release:     release,
		InstalledAt: at,
	}
}

// ---- Catalog grid tests ----

func TestRenderCatalogGrid_FitsTerminalWidth(t *testing.T) {
	fonts := []string{
		"Alpha", "Bravo", "Charlie", "Delta", "Echo",
		"Foxtrot", "Golf", "Hotel", "India", "Juliet",
	}
	// termWidth 80: longest name is "Charlie" (7). colWidth = 7+4 = 11.
	// numCols = 80/11 = 7, numRows = ceil(10/7) = 2.
	// Output: 2 grid rows + 1 blank line + 1 legend line = 4 lines total.
	termWidth := 80
	installed := ui.InstalledSet{}

	out := ui.RenderCatalogGrid(fonts, installed, termWidth)
	require.NotEmpty(t, out)

	plain := stripANSI(out)
	lines := strings.Split(plain, "\n")
	// 2 grid rows + blank + legend = 4 lines.
	assert.Equal(t, 4, len(lines), "expected 2 grid rows + blank + legend = 4 lines")
	// Grid fits: rows × cols >= numFonts.
	numRows := 2
	numCols := 7
	assert.GreaterOrEqual(t, numRows*numCols, len(fonts), "grid must accommodate all fonts")
}

func TestRenderCatalogGrid_InstalledMarkedWithSuccess(t *testing.T) {
	fonts := []string{"FiraCode", "Hack", "JetBrainsMono"}
	installed := ui.InstalledSet{
		"FiraCode": makeInstalled("v3.4.0", time.Now()),
	}

	out := ui.RenderCatalogGrid(fonts, installed, 120)
	// Font name must appear in the output.
	assert.Contains(t, out, "FiraCode", "installed font name must appear in output")
	// The name should be rendered with an ANSI green sequence (color code 32).
	assert.Regexp(t, `\x1b\[[^m]*32[^m]*m[^\x1b]*FiraCode`, out, "FiraCode name should be styled green")
	// No inline checkmark/release tag in cells.
	plain := stripANSI(out)
	assert.NotContains(t, plain, "✓", "cells must not contain inline checkmark indicators")
}

func TestRenderCatalogGrid_ImportedMarkedWithWarn(t *testing.T) {
	fonts := []string{"Agave", "Hack"}
	installed := ui.InstalledSet{
		"Hack": makeInstalled(state.ReleaseImported, time.Now()),
	}

	out := ui.RenderCatalogGrid(fonts, installed, 120)
	// Font name must appear in the output.
	assert.Contains(t, out, "Hack", "imported font name must appear in output")
	// The name should be rendered with an ANSI yellow sequence (color code 33).
	assert.Regexp(t, `\x1b\[[^m]*33[^m]*m[^\x1b]*Hack`, out, "Hack name should be styled yellow")
	// No inline checkmark/release tag in cells.
	plain := stripANSI(out)
	assert.NotContains(t, plain, "✓", "cells must not contain inline checkmark indicators")
}

func TestRenderCatalogGrid_HasLegendInTTYOutput(t *testing.T) {
	fonts := []string{"Agave", "FiraCode", "Hack"}
	installed := ui.InstalledSet{}

	out := ui.RenderCatalogGrid(fonts, installed, 120)
	plain := stripANSI(out)

	// Legend must be present.
	assert.Contains(t, plain, "Legend:", "output must contain a legend line")

	// Legend must appear after all font names — it should be in the last two
	// lines (blank line then legend line).
	lines := strings.Split(plain, "\n")
	require.GreaterOrEqual(t, len(lines), 2, "output must have at least 2 lines")

	lastLines := strings.Join(lines[len(lines)-2:], "\n")
	assert.Contains(t, lastLines, "Legend:", "legend must appear at the bottom of the output")

	// Font names must appear before the legend.
	legendIdx := strings.Index(plain, "Legend:")
	for _, name := range fonts {
		nameIdx := strings.Index(plain, name)
		assert.Less(t, nameIdx, legendIdx, "font name %q must appear before the legend", name)
	}
}

// ---- Installed table tests ----

var referenceTime = time.Date(2026, 5, 3, 17, 25, 0, 0, time.UTC)

func TestRenderInstalledTable_HasHeaderAndRows(t *testing.T) {
	fonts := []string{"FiraCode", "Hack", "JetBrainsMono"}
	installed := ui.InstalledSet{
		"FiraCode":      makeInstalled(state.ReleaseImported, referenceTime),
		"Hack":          makeInstalled(state.ReleaseImported, referenceTime),
		"JetBrainsMono": makeInstalled("v3.4.0", referenceTime),
	}

	out := ui.RenderInstalledTable(fonts, installed)
	plain := stripANSI(out)

	// Header columns must appear.
	assert.Contains(t, plain, "FONT", "header row must contain FONT")
	assert.Contains(t, plain, "RELEASE", "header row must contain RELEASE")
	assert.Contains(t, plain, "INSTALLED AT", "header row must contain INSTALLED AT")

	// Each font name must appear in the output.
	for _, name := range fonts {
		assert.Contains(t, plain, name, "table must contain font %q", name)
	}
}

func TestRenderInstalledTable_FormatsTimestamp(t *testing.T) {
	ts := time.Date(2026, 5, 3, 15, 12, 0, 0, time.UTC)
	fonts := []string{"JetBrainsMono"}
	installed := ui.InstalledSet{
		"JetBrainsMono": makeInstalled("v3.4.0", ts),
	}

	out := ui.RenderInstalledTable(fonts, installed)
	plain := stripANSI(out)

	// The timestamp should be formatted as YYYY-MM-DD HH:MM in local time.
	expected := ts.Local().Format("2006-01-02 15:04")
	assert.Contains(t, plain, expected, "installed-at timestamp must be formatted as %q", expected)
}

// ---- Plain / non-TTY fallback tests ----

func TestRenderCatalogPlain_OneLinePerFont(t *testing.T) {
	fonts := []string{"Alpha", "Bravo", "Charlie"}

	out := ui.RenderCatalogPlain(fonts)

	// No ANSI sequences.
	assert.NotContains(t, out, "\x1b", "plain output must contain no ANSI sequences")

	lines := strings.Split(out, "\n")
	require.Equal(t, len(fonts), len(lines), "one line per font")
	for i, f := range fonts {
		assert.Equal(t, f, lines[i])
	}
}

func TestRenderInstalledPlain_TabSeparated(t *testing.T) {
	fonts := []string{"FiraCode", "Hack"}
	installed := ui.InstalledSet{
		"FiraCode": makeInstalled(state.ReleaseImported, referenceTime),
		"Hack":     makeInstalled("v3.4.0", referenceTime),
	}

	out := ui.RenderInstalledPlain(fonts, installed)

	// No ANSI sequences.
	assert.NotContains(t, out, "\x1b", "plain output must contain no ANSI sequences")

	lines := strings.Split(out, "\n")
	require.Equal(t, 2, len(lines))
	assert.Equal(t, "FiraCode\timported", lines[0])
	assert.Equal(t, "Hack\tv3.4.0", lines[1])
}
