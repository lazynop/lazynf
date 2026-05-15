// Package messages defines the tea.Msg types exchanged between the root app
// and its sub-models. Keeping them in one package avoids import cycles
// between sibling components.
package messages

import (
	"github.com/lazynop/lazynf/internal/engine"
)

// EngineEventMsg wraps an engine.Event so it can flow through bubbletea.
// The OpID is duplicated from Ev.GetOpID() for convenience.
type EngineEventMsg struct {
	OpID engine.OpID
	Ev   engine.Event
}

// OpDoneMsg signals that the engine OpHandle's channel was closed.
// The app removes the OpID from its inFlight map on this message.
type OpDoneMsg struct {
	OpID engine.OpID
}

// FontsLoadedMsg is the initial population (engine.List result). Carries Err
// so the app can render a full-screen error state instead of a fake list.
type FontsLoadedMsg struct {
	Fonts []engine.FontInfo
	Err   error
}

// FontStateChangedMsg patches one font in the fontlist after a CompletedEvent.
type FontStateChangedMsg struct {
	Font engine.FontInfo
}

// FontHighlightedMsg fires when fontlist's cursor moves; detail pane subscribes.
type FontHighlightedMsg struct {
	Font *engine.FontInfo // nil if list is empty
}

// SelectionChangedMsg fires on space/clear; statusbar shows the count.
type SelectionChangedMsg struct {
	Count int
}

// RequestInstallMsg asks the app to launch an install operation.
type RequestInstallMsg struct {
	Tags []string
}

// RequestUpdateMsg asks the app to launch an update operation.
type RequestUpdateMsg struct {
	Tags []string
}

// RequestRemoveMsg asks the app to launch a remove (or purge) operation.
type RequestRemoveMsg struct {
	Tags  []string
	Purge bool
}

// RequestImportMsg asks the app to launch an import operation.
type RequestImportMsg struct {
	Names  []string
	Detect bool
}

// RequestRefreshCatalogMsg asks the app to refresh the GitHub catalog.
type RequestRefreshCatalogMsg struct{}

// RequestDoctorMsg asks the app to launch the doctor flow.
type RequestDoctorMsg struct{}

// DoctorSectionMsg is the TUI-side wrapper of engine.DoctorSectionEvent.
// The doctor pane subscribes.
type DoctorSectionMsg struct {
	OpID    engine.OpID
	Section string
	Title   string
	Status  engine.DoctorStatus
	Detail  string
	Hint    string
	Action  engine.DoctorAction
}

// ConfirmResultMsg is the user's response to a modal confirm.
// Token correlates with the originating Request that opened the modal
// (the app stashes the pending request keyed by token).
type ConfirmResultMsg struct {
	Token  int64
	Choice ConfirmChoice
}

// ConfirmChoice is the discrete answer the modal captured.
type ConfirmChoice int

// ConfirmChoice values returned by the modal.
const (
	ChoiceNo ConfirmChoice = iota
	ChoiceYes
	ChoiceCancel
	ChoiceForce
)

// FocusChangeMsg moves focus to a different pane. Emitted on Tab / numeric
// keys. The app updates its focused field and the panes update their borders.
type FocusChangeMsg struct {
	Pane Pane
}

// Pane identifies which sub-model currently has keyboard focus.
type Pane int

// Pane identifiers for keyboard focus tracking.
const (
	PaneFontlist Pane = iota
	PaneDetail
	PaneLogpane
	PaneDoctor
)
