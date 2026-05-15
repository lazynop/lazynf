package engine

import (
	"context"
	"sync"
)

// pendingResolve tracks per-token resolve channels for ConflictEvents
// currently awaiting the consumer's Resolve call. Thread-safe: the producer
// goroutine writes to the map under mu and waits on the channel; Resolve
// reads the map under mu and writes to the channel.
type pendingResolve struct {
	mu      sync.Mutex
	waiters map[int64]chan ConflictChoice
}

func newPendingResolve() *pendingResolve {
	return &pendingResolve{waiters: map[int64]chan ConflictChoice{}}
}

// resolve sends choice on the channel registered for token. Unknown tokens
// are a silent no-op (e.g. stale resolve after the op already finished).
func (p *pendingResolve) resolve(token int64, choice ConflictChoice) {
	p.mu.Lock()
	ch, ok := p.waiters[token]
	if ok {
		delete(p.waiters, token)
	}
	p.mu.Unlock()
	if ok {
		ch <- choice
	}
}

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
	ch      chan Event
	ctx     context.Context
	pending *pendingResolve
}

func newEmitter(ctx context.Context) *opEmitter {
	return &opEmitter{
		ch:      make(chan Event, 32),
		ctx:     ctx,
		pending: newPendingResolve(),
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

// emitAndWait sends ev as a ConflictEvent and blocks until the consumer
// calls Resolve for the matching token. Returns ChoiceCancel if ctx fires
// before Resolve.
func (e *opEmitter) emitAndWait(ev ConflictEvent) ConflictChoice {
	ch := make(chan ConflictChoice, 1)
	e.pending.mu.Lock()
	e.pending.waiters[ev.Token] = ch
	e.pending.mu.Unlock()

	e.Send(ev)

	select {
	case choice := <-ch:
		return choice
	case <-e.ctx.Done():
		// Clean up the registry to avoid leaking the channel.
		e.pending.mu.Lock()
		delete(e.pending.waiters, ev.Token)
		e.pending.mu.Unlock()
		return ChoiceCancel
	}
}
