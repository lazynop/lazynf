package detail

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/stretchr/testify/require"
)

func TestRender_InstalledFont_ShowsAllFields(t *testing.T) {
	fi := &engine.FontInfo{
		Name:          "FiraCode",
		Status:        engine.StatusInstalled,
		Version:       "v3.2.1",
		LatestVersion: "v3.2.1",
		Files:         []string{"a.ttf", "b.ttf"},
		Size:          1024 * 1024 * 5,
		InstalledAt:   time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
		Dir:           "/home/user/.local/share/fonts/FiraCode",
	}
	m := New()
	m.Current = fi
	m.Width, m.Height = 60, 20
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "FiraCode")
	require.Contains(t, s, "v3.2.1")
	require.Contains(t, s, "5.0 MiB")
	require.Contains(t, s, "/home/user/.local/share/fonts/FiraCode")
}

func TestRender_NilFont_ShowsEmptyState(t *testing.T) {
	m := New()
	m.Width, m.Height = 60, 20
	s := strings.ToLower(ansi.Strip(m.View().Content))
	require.Contains(t, s, "no font selected")
}

func TestInit_ReturnsNil(t *testing.T) {
	require.Nil(t, New().Init())
}

func TestUpdate_FontHighlighted_SetsCurrent(t *testing.T) {
	m := New()
	fi := &engine.FontInfo{Name: "X", Status: engine.StatusAvailable}
	out, cmd := m.Update(messages.FontHighlightedMsg{Font: fi})
	require.Nil(t, cmd)
	require.Equal(t, fi, out.(Model).Current)
}

func TestUpdate_OtherMsg_IsNoOp(t *testing.T) {
	m := New()
	out, cmd := m.Update("ignored")
	require.Nil(t, cmd)
	require.Nil(t, out.(Model).Current)
}

func TestStatusText_AllValues(t *testing.T) {
	require.Contains(t, statusText(engine.StatusInstalled), "installed")
	require.Contains(t, statusText(engine.StatusStale), "update available")
	require.Contains(t, statusText(engine.StatusImported), "imported")
	require.Contains(t, statusText(engine.StatusAvailable), "available")
	require.Contains(t, statusText(engine.StatusUnknown), "unknown")
	// Unknown enum value: falls through to "".
	require.Equal(t, "", statusText(engine.FontStatus(99)))
}

func TestHumanSize_AllUnits(t *testing.T) {
	require.Equal(t, "512 B", humanSize(512))
	require.Equal(t, "1.0 KiB", humanSize(1024))
	require.Equal(t, "2.0 MiB", humanSize(2*1024*1024))
}

func TestRender_ImportedFont_ShowsStatus(t *testing.T) {
	m := New()
	m.Current = &engine.FontInfo{Name: "Imp", Status: engine.StatusImported}
	m.Width, m.Height = 60, 20
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "imported")
}

func TestRender_Unknown_ShowsHint(t *testing.T) {
	m := New()
	m.Current = &engine.FontInfo{Name: "Gone", Status: engine.StatusUnknown}
	m.Width, m.Height = 60, 20
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "unknown")
}

func TestRender_InstalledAtAndDir_PrintLines(t *testing.T) {
	m := New()
	m.Current = &engine.FontInfo{
		Name: "X", Status: engine.StatusInstalled,
		Files: []string{"a.ttf"}, Size: 42,
		InstalledAt: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		Dir:         "/x",
	}
	m.Width, m.Height = 60, 20
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "2026-05-10")
	require.Contains(t, s, "/x")
	require.Contains(t, s, "42 B")
}

func TestRender_StaleFont_ShowsBothVersions(t *testing.T) {
	fi := &engine.FontInfo{
		Name: "X", Status: engine.StatusStale,
		Version: "v3.1.0", LatestVersion: "v3.2.0",
	}
	m := New()
	m.Current = fi
	m.Width, m.Height = 60, 20
	s := ansi.Strip(m.View().Content)
	require.Contains(t, s, "v3.1.0")
	require.Contains(t, s, "v3.2.0")
}
