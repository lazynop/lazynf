// Package drain converts engine.OpHandle channels into tea.Cmd values.
//
// The bubbletea idiom for streaming work is "cmd reads one message, returns
// it, and the Update loop re-arms a new cmd to read the next one." This
// package factors that pattern.
package drain

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
)

// EngineCmd returns a tea.Cmd that reads ONE event from ch and wraps it in
// an EngineEventMsg. When ch is closed it returns OpDoneMsg so the app can
// remove the opID from its inFlight map. The caller re-arms a new EngineCmd
// after handling each EngineEventMsg (do NOT re-arm after OpDoneMsg).
func EngineCmd(opID engine.OpID, ch <-chan engine.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return messages.OpDoneMsg{OpID: opID}
		}
		return messages.EngineEventMsg{OpID: opID, Ev: ev}
	}
}

// EngineCmdCtx is EngineCmd with cancellation support. On ctx.Done it
// behaves as if the channel was closed (returns OpDoneMsg). Use this when
// the app's master ctx may fire mid-op.
func EngineCmdCtx(ctx context.Context, opID engine.OpID, ch <-chan engine.Event) tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-ch:
			if !ok {
				return messages.OpDoneMsg{OpID: opID}
			}
			return messages.EngineEventMsg{OpID: opID, Ev: ev}
		case <-ctx.Done():
			return messages.OpDoneMsg{OpID: opID}
		}
	}
}
