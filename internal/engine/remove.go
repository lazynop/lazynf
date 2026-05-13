package engine

import (
	"context"
	"errors"

	"github.com/lazynop/lazynf/internal/fonts"
)

// RemoveOptions captures user-tunable flags for a Remove call.
type RemoveOptions struct {
	// Purge, if true, also deletes on-disk files for "imported" fonts.
	// For non-imported fonts the flag is a no-op (files are always deleted).
	Purge bool

	// SkipCacheRefresh, if true, suppresses the final fc-cache invocation.
	SkipCacheRefresh bool
}

// Remove launches fonts.Remove for the given tags in a goroutine, translating
// callback events into engine.Event sent on the returned channel. The channel
// closes at termination.
//
// fonts.Remove is local-only (no network), so no retry wrapper.
func (e *Engine) Remove(ctx context.Context, tags []string, opts RemoveOptions) OpHandle {
	opID := e.nextOpID()
	em := newEmitter(ctx)

	go func() {
		defer em.Close()
		em.Send(StartedEvent{OpID: opID, Kind: "remove"})

		params := fonts.RemoveParams{
			Names:     tags,
			StatePath: e.deps.StatePath,
			Refresher: e.deps.FontCache,
		}
		emittedFailures := map[string]bool{}
		fontsOpts := fonts.RemoveOptions{
			Purge:            opts.Purge,
			SkipCacheRefresh: opts.SkipCacheRefresh,
			OnEvent: func(fe fonts.Event) {
				translateRemoveEvent(opID, fe, em.Send, emittedFailures)
			},
		}
		result, err := fonts.Remove(ctx, params, fontsOpts)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				em.Send(CanceledEvent{OpID: opID})
				return
			}
			em.Send(FailedEvent{OpID: opID, Err: err})
			return
		}
		// Surface per-tag failures that fonts.Remove recorded in Result but
		// did not emit via OnEvent (e.g. "not in manifest"). The
		// emittedFailures set deduplicates against any EventRemoveError
		// already routed through the translator.
		if result != nil {
			for tag, ferr := range result.Failures {
				if emittedFailures[tag] {
					continue
				}
				em.Send(FailedEvent{OpID: opID, Target: tag, Err: ferr})
			}
		}
	}()

	return OpHandle{Events: em.Events(), Resolve: noopResolve}
}

// translateRemoveEvent maps fonts.Event to engine.Event for Remove semantics.
//
//	EventRemoveSuccess → CompletedSuccess ("removed")
//	EventRemoveDeadopt → CompletedDeadopted ("manifest entry removed; files left on disk")
//	EventRemoveError   → FailedEvent (and the tag is tracked in emittedFailures
//	                     so the post-call result.Failures sweep does not double-emit).
//	EventCacheRefresh  → StartedEvent{Kind:"fc-cache"}
func translateRemoveEvent(opID OpID, fe fonts.Event, send func(Event), emitted map[string]bool) {
	switch fe.Kind {
	case fonts.EventRemoveSuccess:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedSuccess, Detail: "removed"})
	case fonts.EventRemoveDeadopt:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedDeadopted, Detail: "manifest entry removed; files left on disk"})
	case fonts.EventRemoveError:
		emitted[fe.Font] = true
		send(FailedEvent{OpID: opID, Target: fe.Font, Err: fe.Err})
	case fonts.EventCacheRefresh:
		send(StartedEvent{OpID: opID, Kind: "fc-cache"})
	}
}
