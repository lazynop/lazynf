package engine

import "context"

// opEmitter is the per-operation event sink used by every adapter. It owns
// the channel and bounds emission by ctx cancellation: once ctx.Done fires,
// subsequent Send calls return immediately and the consumer should treat
// channel close as the terminal signal.
//
// The buffer size is intentionally generous (32) so a burst of OnProgress
// callbacks during a download does not block the producer goroutine while
// the consumer is busy rendering a previous event. Adapters that emit far
// fewer events still pay only the small allocation cost.
type opEmitter struct {
	ch  chan Event
	ctx context.Context
}

func newEmitter(ctx context.Context) *opEmitter {
	return &opEmitter{
		ch:  make(chan Event, 32),
		ctx: ctx,
	}
}

// Send delivers ev to the channel unless ctx has already fired, in which
// case the event is dropped silently. Callers should not rely on individual
// events being observable after cancellation — the closing of the channel
// (via Close) is the contract that signals termination.
func (e *opEmitter) Send(ev Event) {
	select {
	case e.ch <- ev:
	case <-e.ctx.Done():
	}
}

// Events exposes the read-only channel for embedding in OpHandle.
func (e *opEmitter) Events() <-chan Event {
	return e.ch
}

// Close terminates the stream. Adapters should defer this.
func (e *opEmitter) Close() {
	close(e.ch)
}
