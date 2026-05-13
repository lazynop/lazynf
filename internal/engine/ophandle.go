package engine

// OpHandle is the return type of all async engine operations.
// The consumer drains Events until the channel closes; Resolve is called in
// response to a ConflictEvent with the received token. Resolve with an unknown
// (e.g. stale) token is a silent no-op.
type OpHandle struct {
	Events  <-chan Event
	Resolve func(token int64, choice ConflictChoice)
}

// noopResolve is the Resolve function for OpHandles that do not (yet) emit
// ConflictEvent. Tasks 6-10 reuse this. Plan 2 will replace with real
// resolution for adapters that emit conflicts.
func noopResolve(_ int64, _ ConflictChoice) {}
