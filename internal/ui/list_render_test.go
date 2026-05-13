package ui_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lazynop/lazynf/internal/engine"
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

// available builds a FontInfo for a font present in the catalog only.
func available(name string) engine.FontInfo {
	return engine.FontInfo{Name: name, Status: engine.StatusAvailable}
}

// installed builds a FontInfo for a font installed at a real release tag.
func installed(name, version string, at time.Time) engine.FontInfo {
	return engine.FontInfo{
		Name:          name,
		Status:        engine.StatusInstalled,
		Version:       version,
		LatestVersion: version,
		InstalledAt:   at,
	}
}

// imported builds a FontInfo for a font adopted via `lazynf import` without
// version detection.
func imported(name string, at time.Time) engine.FontInfo {
	return engine.FontInfo{
		Name:        name,
		Status:      engine.StatusImported,
		InstalledAt: at,
	}
}

// ---- Catalog grid tests ----

func TestRenderCatalogGrid_FitsTerminalWidth(t *testing.T) {
	names := []string{
		"Alpha", "Bravo", "Charlie", "Delta", "Echo",
		"Foxtrot", "Golf", "Hotel", "India", "Juliet",
	}
	infos := make([]engine.FontInfo, len(names))
	for i, n := range names {
		infos[i] = available(n)
	}
	// termWidth 80: longest name is "Charlie" (7). colWidth = 7+4 = 11.
	// numCols = 80/11 = 7, numRows = ceil(10/7) = 2.
	// Output: 2 grid rows + 1 blank line + 1 legend line = 4 lines total.
	termWidth := 80

	out := ui.RenderCatalogGrid(infos, termWidth)
	require.NotEmpty(t, out)

	plain := stripANSI(out)
	lines := strings.Split(plain, "\n")
	// 2 grid rows + blank + legend = 4 lines.
	assert.Equal(t, 4, len(lines), "expected 2 grid rows + blank + legend = 4 lines")
	// Grid fits: rows × cols >= numFonts.
	numRows := 2
	numCols := 7
	assert.GreaterOrEqual(t, numRows*numCols, len(infos), "grid must accommodate all fonts")
}

func TestRenderCatalogGrid_InstalledMarkedWithSuccess(t *testing.T) {
	infos := []engine.FontInfo{
		installed("FiraCode", "v3.4.0", time.Now()),
		available("Hack"),
		available("JetBrainsMono"),
	}

	out := ui.RenderCatalogGrid(infos, 120)
	// Font name must appear in the output.
	assert.Contains(t, out, "FiraCode", "installed font name must appear in output")
	// The name should be rendered with an ANSI green sequence (color code 32).
	assert.Regexp(t, `\x1b\[[^m]*32[^m]*m[^\x1b]*FiraCode`, out, "FiraCode name should be styled green")
	// No inline checkmark/release tag in cells.
	plain := stripANSI(out)
	assert.NotContains(t, plain, "✓", "cells must not contain inline checkmark indicators")
}

func TestRenderCatalogGrid_ImportedMarkedWithWarn(t *testing.T) {
	infos := []engine.FontInfo{
		available("Agave"),
		imported("Hack", time.Now()),
	}

	out := ui.RenderCatalogGrid(infos, 120)
	// Font name must appear in the output.
	assert.Contains(t, out, "Hack", "imported font name must appear in output")
	// The name should be rendered with an ANSI yellow sequence (color code 33).
	assert.Regexp(t, `\x1b\[[^m]*33[^m]*m[^\x1b]*Hack`, out, "Hack name should be styled yellow")
	// No inline checkmark/release tag in cells.
	plain := stripANSI(out)
	assert.NotContains(t, plain, "✓", "cells must not contain inline checkmark indicators")
}

func TestRenderCatalogGrid_HasLegendInTTYOutput(t *testing.T) {
	names := []string{"Agave", "FiraCode", "Hack"}
	infos := make([]engine.FontInfo, len(names))
	for i, n := range names {
		infos[i] = available(n)
	}

	out := ui.RenderCatalogGrid(infos, 120)
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
	for _, n := range names {
		nameIdx := strings.Index(plain, n)
		assert.Less(t, nameIdx, legendIdx, "font name %q must appear before the legend", n)
	}
}

// ---- Installed table tests ----

var referenceTime = time.Date(2026, 5, 3, 17, 25, 0, 0, time.UTC)

func TestRenderInstalledTable_HasHeaderAndRows(t *testing.T) {
	infos := []engine.FontInfo{
		imported("FiraCode", referenceTime),
		imported("Hack", referenceTime),
		installed("JetBrainsMono", "v3.4.0", referenceTime),
	}

	out := ui.RenderInstalledTable(infos)
	plain := stripANSI(out)

	// Header columns must appear.
	assert.Contains(t, plain, "FONT", "header row must contain FONT")
	assert.Contains(t, plain, "RELEASE", "header row must contain RELEASE")
	assert.Contains(t, plain, "INSTALLED AT", "header row must contain INSTALLED AT")

	// Each font name must appear in the output.
	for _, fi := range infos {
		assert.Contains(t, plain, fi.Name, "table must contain font %q", fi.Name)
	}
}

func TestRenderInstalledTable_FormatsTimestamp(t *testing.T) {
	ts := time.Date(2026, 5, 3, 15, 12, 0, 0, time.UTC)
	infos := []engine.FontInfo{
		installed("JetBrainsMono", "v3.4.0", ts),
	}

	out := ui.RenderInstalledTable(infos)
	plain := stripANSI(out)

	// The timestamp should be formatted as YYYY-MM-DD HH:MM in local time.
	expected := ts.Local().Format("2006-01-02 15:04")
	assert.Contains(t, plain, expected, "installed-at timestamp must be formatted as %q", expected)
}

// ---- Plain / non-TTY fallback tests ----

func TestRenderCatalogPlain_OneLinePerFont(t *testing.T) {
	names := []string{"Alpha", "Bravo", "Charlie"}
	infos := []engine.FontInfo{available("Alpha"), available("Bravo"), available("Charlie")}

	out := ui.RenderCatalogPlain(infos)

	// No ANSI sequences.
	assert.NotContains(t, out, "\x1b", "plain output must contain no ANSI sequences")

	lines := strings.Split(out, "\n")
	require.Equal(t, len(infos), len(lines), "one line per font")
	for i, n := range names {
		assert.Equal(t, n, lines[i])
	}
}

func TestRenderInstalledPlain_TabSeparated(t *testing.T) {
	infos := []engine.FontInfo{
		imported("FiraCode", referenceTime),
		installed("Hack", "v3.4.0", referenceTime),
	}

	out := ui.RenderInstalledPlain(infos)

	// No ANSI sequences.
	assert.NotContains(t, out, "\x1b", "plain output must contain no ANSI sequences")

	lines := strings.Split(out, "\n")
	require.Equal(t, 2, len(lines))
	assert.Equal(t, "FiraCode\t"+state.ReleaseImported, lines[0])
	assert.Equal(t, "Hack\tv3.4.0", lines[1])
}
