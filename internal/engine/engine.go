package engine

import (
	"sync/atomic"
	"time"
)

// Engine orchestrates font operations exposing an event-driven API.
// It is stateless beyond OpID/Token counters: all domain state (manifest,
// catalog) is read/written on each operation via Deps.
type Engine struct {
	deps   Deps
	opCtr  atomic.Int64
	tokCtr atomic.Int64
}

// New constructs an Engine. Default values are applied on Deps.
func New(deps Deps) *Engine {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &Engine{deps: deps}
}

// nextOpID returns a unique, monotonic OpID.
func (e *Engine) nextOpID() OpID {
	return OpID(e.opCtr.Add(1))
}

// nextToken returns a unique token for ConflictEvent.Resolve.
func (e *Engine) nextToken() int64 {
	return e.tokCtr.Add(1)
}
