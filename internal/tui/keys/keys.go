// Package keys centralises every key binding used by the TUI.
// The help overlay reads from this struct to render itself.
package keys

import "charm.land/bubbles/v2/key"

// KeyMap groups all bindings. Global bindings work in every pane; per-pane
// bindings shadow globals only where it makes sense (e.g. Filter shadows
// fontlist's local Filter binding).
type KeyMap struct {
	// Global
	Quit      key.Binding
	Help      key.Binding
	FocusNext key.Binding // Tab
	FocusPrev key.Binding // Shift+Tab
	Doctor    key.Binding // d
	Refresh   key.Binding // R — RefreshCatalog

	// Fontlist
	Up          key.Binding
	Down        key.Binding
	Top         key.Binding // g
	Bottom      key.Binding // G
	Filter      key.Binding // /
	ClearFilter key.Binding // esc when filter focused
	SortCycle   key.Binding // s
	Select      key.Binding // space — toggle selection
	ClearSelect key.Binding // esc when no filter
	Install     key.Binding // i
	Update      key.Binding // u
	Remove      key.Binding // r
	Purge       key.Binding // P (capital — destructive)
	Import      key.Binding // I (capital — import flow)
	ToggleLog   key.Binding // L

	// Modal
	ConfirmYes    key.Binding
	ConfirmNo     key.Binding
	ConfirmForce  key.Binding
	ConfirmCancel key.Binding
}

// Default returns the canonical bindings; future custom-config can override.
func Default() KeyMap {
	return KeyMap{
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		FocusNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
		FocusPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev pane")),
		Doctor:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "doctor")),
		Refresh:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh catalog")),

		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Top:         key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
		Bottom:      key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		Filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ClearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
		SortCycle:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Select:      key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
		ClearSelect: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear selection")),
		Install:     key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
		Update:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update")),
		Remove:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "remove")),
		Purge:       key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "purge (destructive)")),
		Import:      key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "import")),
		ToggleLog:   key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "toggle log pane")),

		ConfirmYes:    key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y/enter", "yes")),
		ConfirmNo:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "no")),
		ConfirmForce:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "force")),
		ConfirmCancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

// ShortHelp returns the minimal hints to show in the statusbar by default.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Filter, k.Install, k.Remove, k.Help, k.Quit}
}

// FullHelp returns the full grid for the help overlay (rows of related binds).
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.Filter, k.ClearFilter, k.SortCycle, k.Select},
		{k.Install, k.Update, k.Remove, k.Purge, k.Import},
		{k.Doctor, k.Refresh, k.ToggleLog},
		{k.FocusNext, k.FocusPrev, k.Help, k.Quit},
	}
}
