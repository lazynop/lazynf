package ui_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lazynop/vellum/internal/state"
	"github.com/lazynop/vellum/internal/ui"
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
	// termWidth 80: longest name is "Charlie" (7). colWidth = 7+12+4 = 23.
	// numCols = 80/23 = 3.
	termWidth := 80
	installed := ui.InstalledSet{}

	out := ui.RenderCatalogGrid(fonts, installed, termWidth)
	require.NotEmpty(t, out)

	plain := stripANSI(out)
	lines := strings.Split(plain, "\n")
	// With 10 fonts and 3 cols: numRows = ceil(10/3) = 4.
	assert.Equal(t, 4, len(lines), "expected 4 rows for 10 fonts in 3 columns")
}

func TestRenderCatalogGrid_InstalledMarkedWithSuccess(t *testing.T) {
	fonts := []string{"FiraCode", "Hack", "JetBrainsMono"}
	installed := ui.InstalledSet{
		"FiraCode": makeInstalled("v3.4.0", time.Now()),
	}

	out := ui.RenderCatalogGrid(fonts, installed, 120)
	// The checkmark and release tag must appear somewhere in the output.
	assert.Contains(t, out, "✓v3.4.0", "installed font should show ✓ + release")
	// Hack and JetBrainsMono are not installed — no checkmark for them.
	plain := stripANSI(out)
	// Count occurrences of ✓ — should be exactly 1.
	assert.Equal(t, 1, strings.Count(plain, "✓"), "only one font is installed")
}

func TestRenderCatalogGrid_ImportedMarkedWithWarn(t *testing.T) {
	fonts := []string{"Agave", "Hack"}
	installed := ui.InstalledSet{
		"Hack": makeInstalled(state.ReleaseImported, time.Now()),
	}

	out := ui.RenderCatalogGrid(fonts, installed, 120)
	assert.Contains(t, out, "✓imported", "imported font should show ✓imported")

	// Verify the indicator is present exactly once.
	plain := stripANSI(out)
	assert.Equal(t, 1, strings.Count(plain, "✓"), "only one font is marked")
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
