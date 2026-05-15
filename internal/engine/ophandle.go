package engine

// OpHandle is the consumer-side handle for an in-flight engine operation.
// Events delivers all engine events for the op; Resolve is called in response
// to a ConflictEvent with the matching token. Resolve with an unknown or
// stale token is a silent no-op.
type OpHandle struct {
	Events  <-chan Event
	Resolve func(token int64, choice ConflictChoice)
}
