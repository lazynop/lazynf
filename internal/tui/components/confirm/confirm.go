// Package confirm shows a yes/no/cancel/force modal and emits ConfirmResultMsg.
package confirm

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/lazynop/lazynf/internal/tui/theme"
)

// Package-level styles hoisted out of View to avoid per-frame allocation.
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.TextHi)
	dim        = lipgloss.NewStyle().Foreground(theme.TextDim)
)

// Model is a centered modal that asks a yes/no question (optionally with a
// "force" third choice for destructive ops).
type Model struct {
	// Keys is the bound KeyMap (ConfirmYes/No/Cancel/Force live there).
	Keys keys.KeyMap
	// Token correlates the result with the originating Request.
	Token int64
	// Title is the bold first line of the modal.
	Title string
	// Body is the explanatory text shown under the title.
	Body string

	// AllowForce makes 'f' available as a third choice (destructive ops).
	AllowForce bool
	// AllowAdopt makes 'a' available as an extra choice for FilesOnDisk
	// conflicts (register the on-disk files in the manifest).
	AllowAdopt bool

	// Width is the parent terminal width used for centering.
	Width int
	// Height is the parent terminal height used for centering.
	Height int
}

// New constructs a confirm modal for the given token + title + body.
func New(k keys.KeyMap, token int64, title, body string) Model {
	return Model{Keys: k, Token: token, Title: title, Body: body}
}

// Init is a no-op (the modal is purely reactive to keypresses).
func (m Model) Init() tea.Cmd { return nil }

// Update consumes the user's key press and emits a ConfirmResultMsg.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kmsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(kmsg, m.Keys.ConfirmYes):
		return m, messages.Cmd(messages.ConfirmResultMsg{Token: m.Token, Choice: messages.ChoiceYes})
	case key.Matches(kmsg, m.Keys.ConfirmNo):
		return m, messages.Cmd(messages.ConfirmResultMsg{Token: m.Token, Choice: messages.ChoiceNo})
	case m.AllowForce && key.Matches(kmsg, m.Keys.ConfirmForce):
		return m, messages.Cmd(messages.ConfirmResultMsg{Token: m.Token, Choice: messages.ChoiceForce})
	case m.AllowAdopt && key.Matches(kmsg, m.Keys.ConfirmAdopt):
		return m, messages.Cmd(messages.ConfirmResultMsg{Token: m.Token, Choice: messages.ChoiceAdopt})
	case key.Matches(kmsg, m.Keys.ConfirmCancel):
		return m, messages.Cmd(messages.ConfirmResultMsg{Token: m.Token, Choice: messages.ChoiceCancel})
	}
	return m, nil
}

// View renders the centered modal.
func (m Model) View() tea.View {
	hints := dim.Render("y/enter: yes  n: no  esc: cancel")
	switch {
	case m.AllowForce && m.AllowAdopt:
		hints = dim.Render("y/enter: yes  n: no  f: force  a: adopt  esc: cancel")
	case m.AllowForce:
		hints = dim.Render("y/enter: yes  n: no  f: force  esc: cancel")
	case m.AllowAdopt:
		hints = dim.Render("y/enter: yes  n: no  a: adopt  esc: cancel")
	}

	body := titleStyle.Render(m.Title) + "\n\n" + m.Body + "\n\n" + hints
	box := theme.PaneStyle(true).Padding(1, 2).Render(body)

	w, h := m.Width, m.Height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return tea.NewView(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box))
}
