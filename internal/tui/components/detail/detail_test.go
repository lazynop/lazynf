package detail

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/lazynop/lazynf/internal/engine"
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
