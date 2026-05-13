package engine

import (
	"testing"
	"time"
)

// DrainEvents is a test helper that empties an OpHandle until the channel
// closes (or a timeout fires) and returns all observed events.
// Do NOT use in production code — the preferred pattern is a `range` loop
// with a type switch.
func DrainEvents(t *testing.T, h OpHandle) []Event {
	t.Helper()
	var got []Event
	timeout := time.After(10 * time.Second)
	for {
		select {
		case ev, ok := <-h.Events:
			if !ok {
				return got
			}
			got = append(got, ev)
		case <-timeout:
			t.Fatalf("DrainEvents: timeout after 10s; got %d events so far", len(got))
		}
	}
}
