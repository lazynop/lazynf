package drain

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/stretchr/testify/require"
)

func TestEngineCmd_OneEvent_ReturnsEngineEventMsg(t *testing.T) {
	ch := make(chan engine.Event, 1)
	ch <- engine.StartedEvent{OpID: 7, Kind: "test"}

	cmd := EngineCmd(7, ch)
	msg := cmd()

	got, ok := msg.(messages.EngineEventMsg)
	require.True(t, ok, "expected EngineEventMsg, got %T", msg)
	require.Equal(t, engine.OpID(7), got.OpID)
	require.IsType(t, engine.StartedEvent{}, got.Ev)
}

func TestEngineCmd_ChannelClosed_ReturnsOpDoneMsg(t *testing.T) {
	ch := make(chan engine.Event)
	close(ch)

	cmd := EngineCmd(42, ch)
	msg := cmd()

	got, ok := msg.(messages.OpDoneMsg)
	require.True(t, ok, "expected OpDoneMsg, got %T", msg)
	require.Equal(t, engine.OpID(42), got.OpID)
}

func TestEngineCmd_BlocksUntilEvent(t *testing.T) {
	ch := make(chan engine.Event)
	cmd := EngineCmd(1, ch)

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	select {
	case <-done:
		t.Fatal("cmd returned before any event was sent")
	case <-time.After(20 * time.Millisecond):
	}

	ch <- engine.CompletedEvent{OpID: 1, Kind: engine.CompletedSuccess}
	select {
	case msg := <-done:
		require.IsType(t, messages.EngineEventMsg{}, msg)
	case <-time.After(time.Second):
		t.Fatal("cmd did not return after event sent")
	}
}

func TestEngineCmd_RespectsCtxCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan engine.Event)
	cmd := EngineCmdCtx(ctx, 1, ch)

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	cancel()
	select {
	case msg := <-done:
		_, ok := msg.(messages.OpDoneMsg)
		require.True(t, ok, "expected OpDoneMsg on ctx cancel, got %T", msg)
	case <-time.After(time.Second):
		t.Fatal("cmd did not return after ctx cancel")
	}
}
