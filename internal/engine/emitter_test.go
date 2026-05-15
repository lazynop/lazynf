package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmitter_SendBeforeCtxDone(t *testing.T) {
	em := newEmitter(context.Background())
	em.Send(StartedEvent{OpID: 1})
	em.Close()
	got := []Event{}
	for ev := range em.Events() {
		got = append(got, ev)
	}
	require.Len(t, got, 1)
}

func TestEmitter_SendAfterCtxDoneDoesNotBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	em := newEmitter(ctx)
	cancel()
	// Fill the buffer; Send should drop silently after that, not block.
	for i := 0; i < 100; i++ {
		em.Send(StartedEvent{OpID: 1})
	}
	em.Close()
}

func TestEmitter_EmitAndWait_ReturnsResolvedChoice(t *testing.T) {
	em := newEmitter(context.Background())
	defer em.Close()

	done := make(chan ConflictChoice, 1)
	go func() {
		done <- em.emitAndWait(ConflictEvent{OpID: 1, Token: 7, Kind: ConflictAlreadyImported})
	}()

	// Drain the emitted ConflictEvent so the consumer side has "seen" it.
	<-em.Events()
	em.pending.resolve(7, ChoiceForce)
	require.Equal(t, ChoiceForce, <-done)
}

func TestEmitter_EmitAndWait_CtxCancel_ReturnsChoiceCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	em := newEmitter(ctx)
	defer em.Close()

	done := make(chan ConflictChoice, 1)
	go func() {
		done <- em.emitAndWait(ConflictEvent{OpID: 1, Token: 9, Kind: ConflictFilesOnDisk})
	}()
	<-em.Events()
	cancel()
	require.Equal(t, ChoiceCancel, <-done)
	// Waiter map should be cleaned up so the channel doesn't leak.
	em.pending.mu.Lock()
	_, leaked := em.pending.waiters[9]
	em.pending.mu.Unlock()
	require.False(t, leaked)
}

func TestPendingResolve_UnknownToken_NoOp(t *testing.T) {
	p := newPendingResolve()
	// Should not panic; nothing to deliver to.
	p.resolve(99, ChoiceCancel)
}
