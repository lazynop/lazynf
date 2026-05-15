package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
)

// TestIntegration_ConflictEvent_OpensModalAndResolves drives the app through a
// ConflictEvent and a subsequent ConfirmResultMsg, asserting that the modal
// opens, the token-to-op mapping is recorded, and the matching OpHandle.Resolve
// is called with the translated engine choice.
func TestIntegration_ConflictEvent_OpensModalAndResolves(t *testing.T) {
	eng := engine.New(engine.Deps{})
	m := New(eng)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Inject a fake OpHandle into inFlight so handleConfirmResult can resolve.
	var resolvedToken int64
	var resolvedChoice engine.ConflictChoice
	m.inFlight[7] = engine.OpHandle{
		Events: make(chan engine.Event),
		Resolve: func(token int64, c engine.ConflictChoice) {
			resolvedToken = token
			resolvedChoice = c
		},
	}

	// Simulate engine emitting a ConflictEvent for opID=7, token=42.
	conflict := engine.ConflictEvent{
		OpID:    7,
		Target:  "FiraCode",
		Kind:    engine.ConflictAlreadyImported,
		Choices: []engine.ConflictChoice{engine.ChoiceSkip, engine.ChoiceForce},
		Token:   42,
	}
	out, _ := m.Update(messages.EngineEventMsg{OpID: 7, Ev: conflict})
	mm := out.(*Model)
	require.Equal(t, OverlayConfirm, mm.overlay, "expected confirm overlay after ConflictEvent")
	require.Equal(t, engine.OpID(7), mm.pendingConflict[42], "pendingConflict must map token to opID")

	// User picks "Yes" — translates to ChoiceForce.
	out, _ = mm.Update(messages.ConfirmResultMsg{Token: 42, Choice: messages.ChoiceYes})
	mm = out.(*Model)
	require.Equal(t, OverlayNone, mm.overlay, "modal closes after resolve")
	require.Equal(t, int64(42), resolvedToken, "Resolve called with correct token")
	require.Equal(t, engine.ChoiceForce, resolvedChoice, "ChoiceYes translates to ChoiceForce")
	require.NotContains(t, mm.pendingConflict, int64(42), "token cleared from registry")
}

// TestIntegration_ConflictEvent_Cancel_TranslatesToSkip verifies that a
// ChoiceCancel from the modal becomes engine.ChoiceSkip on the wire.
func TestIntegration_ConflictEvent_Cancel_TranslatesToSkip(t *testing.T) {
	eng := engine.New(engine.Deps{})
	m := New(eng)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	var got engine.ConflictChoice
	m.inFlight[3] = engine.OpHandle{
		Events:  make(chan engine.Event),
		Resolve: func(_ int64, c engine.ConflictChoice) { got = c },
	}

	_, _ = m.Update(messages.EngineEventMsg{OpID: 3, Ev: engine.ConflictEvent{
		OpID: 3, Target: "X", Kind: engine.ConflictAlreadyImported,
		Choices: []engine.ConflictChoice{engine.ChoiceSkip, engine.ChoiceForce}, Token: 99,
	}})
	_, _ = m.Update(messages.ConfirmResultMsg{Token: 99, Choice: messages.ChoiceCancel})
	require.Equal(t, engine.ChoiceSkip, got)
}
