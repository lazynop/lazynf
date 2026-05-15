// Package fontlist renders the left-hand list with cursor + filter + sort
// + multi-select.
package fontlist

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/lazynop/lazynf/internal/tui/theme"
)

// SortKey enumerates the sort orderings the user can cycle through.
type SortKey int

const (
	// SortByName sorts the visible list alphabetically by font name.
	SortByName SortKey = iota
	// SortByStatus groups fonts by their FontStatus enum order.
	SortByStatus
	// SortBySize sorts installed fonts descending by on-disk size.
	SortBySize
)

// Model is the fontlist state.
type Model struct {
	Keys keys.KeyMap

	fonts    []engine.FontInfo
	filter   string
	sort     SortKey
	cursor   int
	selected map[string]bool
	offset   int

	Width, Height int
	Focused       bool

	// FilterEditing is true while the user types into the filter buffer.
	FilterEditing bool
}

// New constructs an empty fontlist with the given KeyMap.
func New(k keys.KeyMap) Model {
	return Model{Keys: k, selected: map[string]bool{}}
}

// Init is a no-op (state is populated by FontsLoadedMsg).
func (m Model) Init() tea.Cmd { return nil }

// Update routes incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case messages.FontsLoadedMsg:
		m.fonts = x.Fonts
		m.cursor = 0
		m.offset = 0
		return m, m.emitHighlight()

	case messages.FontStateChangedMsg:
		for i, f := range m.fonts {
			if f.Name == x.Font.Name {
				m.fonts[i] = x.Font
				return m, nil
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.FilterEditing {
			return m.handleFilterKey(x)
		}
		return m.handleKey(x)
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	visible := m.Visible()
	switch {
	case key.Matches(k, m.Keys.Down):
		if m.cursor < len(visible)-1 {
			m.cursor++
		}
		return m, m.emitHighlight()
	case key.Matches(k, m.Keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, m.emitHighlight()
	case key.Matches(k, m.Keys.Top):
		m.cursor = 0
		return m, m.emitHighlight()
	case key.Matches(k, m.Keys.Bottom):
		m.cursor = maxInt(0, len(visible)-1)
		return m, m.emitHighlight()
	case key.Matches(k, m.Keys.Filter):
		m.FilterEditing = true
		return m, nil
	case key.Matches(k, m.Keys.SortCycle):
		m.sort = (m.sort + 1) % 3
		return m, nil
	case key.Matches(k, m.Keys.Select):
		if cur := m.cursorFont(visible); cur != nil {
			if m.selected[cur.Name] {
				delete(m.selected, cur.Name)
			} else {
				m.selected[cur.Name] = true
			}
		}
		return m, sendMsg(messages.SelectionChangedMsg{Count: len(m.selected)})
	case key.Matches(k, m.Keys.ClearSelect):
		m.selected = map[string]bool{}
		return m, sendMsg(messages.SelectionChangedMsg{Count: 0})
	case key.Matches(k, m.Keys.Install):
		return m, sendMsg(messages.RequestInstallMsg{Tags: m.targets(visible)})
	case key.Matches(k, m.Keys.Update):
		return m, sendMsg(messages.RequestUpdateMsg{Tags: m.targets(visible)})
	case key.Matches(k, m.Keys.Remove):
		return m, sendMsg(messages.RequestRemoveMsg{Tags: m.targets(visible), Purge: false})
	case key.Matches(k, m.Keys.Purge):
		return m, sendMsg(messages.RequestRemoveMsg{Tags: m.targets(visible), Purge: true})
	case key.Matches(k, m.Keys.Import):
		return m, sendMsg(messages.RequestImportMsg{Names: m.targets(visible), Detect: true})
	}
	return m, nil
}

func (m Model) handleFilterKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(k, m.Keys.ClearFilter) {
		m.filter = ""
		m.FilterEditing = false
		m.cursor = 0
		return m, m.emitHighlight()
	}
	if k.Code == tea.KeyEnter {
		m.FilterEditing = false
		return m, nil
	}
	if k.Code == tea.KeyBackspace {
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
		}
		return m, m.emitHighlight()
	}
	if r := k.Code; r >= 0x20 && r <= 0x7e {
		m.filter += string(rune(r))
		m.cursor = 0
		return m, m.emitHighlight()
	}
	return m, nil
}

// Visible returns the filtered + sorted view of fonts.
func (m Model) Visible() []engine.FontInfo {
	out := make([]engine.FontInfo, 0, len(m.fonts))
	q := strings.ToLower(m.filter)
	for _, f := range m.fonts {
		if q == "" || strings.Contains(strings.ToLower(f.Name), q) {
			out = append(out, f)
		}
	}
	switch m.sort {
	case SortByName:
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	case SortByStatus:
		sort.Slice(out, func(i, j int) bool { return out[i].Status < out[j].Status })
	case SortBySize:
		sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	}
	return out
}

func (m Model) cursorFont(visible []engine.FontInfo) *engine.FontInfo {
	if len(visible) == 0 || m.cursor >= len(visible) {
		return nil
	}
	f := visible[m.cursor]
	return &f
}

// targets returns the selection if any, else the cursor row.
func (m Model) targets(visible []engine.FontInfo) []string {
	if len(m.selected) > 0 {
		out := make([]string, 0, len(m.selected))
		for n := range m.selected {
			out = append(out, n)
		}
		sort.Strings(out)
		return out
	}
	if cur := m.cursorFont(visible); cur != nil {
		return []string{cur.Name}
	}
	return nil
}

func (m Model) emitHighlight() tea.Cmd {
	cur := m.cursorFont(m.Visible())
	return sendMsg(messages.FontHighlightedMsg{Font: cur})
}

// View renders the list pane.
func (m Model) View() tea.View {
	visible := m.Visible()
	rows := make([]string, 0, len(visible)+1)
	for i, f := range visible {
		rows = append(rows, renderRow(f, m.selected[f.Name], i == m.cursor))
	}
	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(theme.TextDim).Render("(no fonts)"))
	}

	body := strings.Join(rows, "\n")
	if m.FilterEditing {
		body = "/" + m.filter + "\n" + body
	}

	border := theme.PaneStyle(m.Focused).
		Width(m.Width).
		Height(m.Height).
		Padding(0, 1)
	return tea.NewView(border.Render(body))
}

func renderRow(f engine.FontInfo, selected, cursor bool) string {
	check := "  "
	if selected {
		check = theme.CheckedRow().Render("◉ ")
	}
	statusGlyph := "  "
	switch f.Status {
	case engine.StatusInstalled:
		statusGlyph = theme.SymbolOK() + " "
	case engine.StatusStale:
		statusGlyph = theme.SymbolWarn() + " "
	case engine.StatusImported:
		statusGlyph = "○ "
	}
	line := check + statusGlyph + f.Name
	if cursor {
		return theme.SelectedRow().Render(line)
	}
	return line
}

func sendMsg(m tea.Msg) tea.Cmd { return func() tea.Msg { return m } }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
