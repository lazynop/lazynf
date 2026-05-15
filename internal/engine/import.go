package engine

import (
	"context"
	"errors"

	"github.com/lazynop/lazynf/internal/fonts"
	"github.com/lazynop/lazynf/internal/state"
)

// ImportOptions captures user-tunable flags for an Import call.
type ImportOptions struct {
	// All scans FontDir for matching Nerd Font sub-dirs when Names is empty.
	All bool

	// Detect hashes files against the latest release to identify the actual
	// installed version; on mismatch the entry falls back to the "imported"
	// sentinel.
	Detect bool

	// Force re-imports even if the font is already in the manifest.
	Force bool
}

// Import launches fonts.Import for the given names (empty + All=true scans
// FontDir) in a goroutine, translating callback events into engine.Event sent
// on the returned channel. The channel closes at termination.
//
// fonts.Import may hit the network only when Detect is true (to fetch release
// asset hashes); the whole call is wrapped in retry so transient timeouts
// during detection are retried.
func (e *Engine) Import(ctx context.Context, names []string, opts ImportOptions) OpHandle {
	opID := e.nextOpID()
	em := newEmitter(ctx)

	go func() {
		defer em.Close()
		em.Send(StartedEvent{OpID: opID, Kind: KindImport})

		// Pre-flight: surface AlreadyImported conflicts so the consumer can
		// opt-in to a force re-import. With Force=true the engine skips the
		// check and proceeds straight to fonts.Import (silent overwrite).
		if !opts.Force && len(names) > 0 {
			manifest, mErr := state.Load(e.deps.StatePath)
			if mErr == nil {
				for _, name := range names {
					entry, ok := manifest.Installed[name]
					if ok && entry.IsImported() {
						choice := em.emitAndWait(ConflictEvent{
							OpID:    opID,
							Target:  name,
							Kind:    ConflictAlreadyImported,
							Choices: []ConflictChoice{ChoiceSkip, ChoiceForce},
							Token:   e.nextToken(),
						})
						switch choice {
						case ChoiceSkip, ChoiceCancel:
							em.Send(CanceledEvent{OpID: opID, Target: name})
							return
						case ChoiceForce:
							opts.Force = true
						}
						// Only one conflict prompt per import call; rest of the batch
						// continues under the resolved opts.
						break
					}
				}
			}
		}

		params := fonts.ImportParams{
			Names:        names,
			All:          opts.All,
			Detect:       opts.Detect,
			Force:        opts.Force,
			FontDir:      e.deps.FontDir,
			StatePath:    e.deps.StatePath,
			CatalogPath:  e.deps.CatalogPath,
			AssetURLBase: e.deps.AssetURLBase,
			GitHub:       e.deps.GitHub,
		}
		// Tracks per-font failures already surfaced via OnEvent so the
		// post-call sweep of result.Failures doesn't double-emit.
		emittedFailures := map[string]struct{}{}
		fontsOpts := fonts.ImportOptions{
			OnEvent: func(fe fonts.Event) {
				translateImportEvent(opID, fe, em.Send, emittedFailures)
			},
		}

		var result *fonts.ImportResult
		err := retry(ctx, func() error {
			r, ferr := fonts.Import(ctx, params, fontsOpts)
			result = r
			return ferr
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				em.Send(CanceledEvent{OpID: opID})
				return
			}
			em.Send(FailedEvent{OpID: opID, Err: err, Retriable: isRetriableNetErr(err)})
			return
		}
		// Surface per-name failures recorded in the result that did not flow
		// through OnEvent. The dedup set guarantees each failure is emitted
		// at most once.
		if result != nil {
			for tag, ferr := range result.Failures {
				if _, dup := emittedFailures[tag]; dup {
					continue
				}
				em.Send(FailedEvent{OpID: opID, Target: tag, Err: ferr})
			}
		}
	}()

	return OpHandle{
		Events:  em.Events(),
		Resolve: func(token int64, choice ConflictChoice) { em.pending.resolve(token, choice) },
	}
}

// translateImportEvent maps fonts.Event to engine.Event for Import semantics.
//
//	EventImportStart   → LogEvent "importing"
//	EventImportSuccess → CompletedSuccess ("imported")
//	EventImportSkipped → CompletedSkipped ("already imported")
//	EventImportError   → FailedEvent (and the tag is tracked in emitted so the
//	                     post-call result.Failures sweep does not double-emit).
func translateImportEvent(opID OpID, fe fonts.Event, send func(Event), emitted map[string]struct{}) {
	switch fe.Kind {
	case fonts.EventImportStart:
		send(LogEvent{OpID: opID, Target: fe.Font, Level: LevelInfo, Message: "importing"})
	case fonts.EventImportSuccess:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedSuccess, Detail: "imported"})
	case fonts.EventImportSkipped:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedSkipped, Detail: "already imported"})
	case fonts.EventImportError:
		emitted[fe.Font] = struct{}{}
		send(FailedEvent{OpID: opID, Target: fe.Font, Err: fe.Err})
	}
}
