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
