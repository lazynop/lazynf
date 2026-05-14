package engine

import (
	"context"
	"errors"

	"github.com/lazynop/lazynf/internal/fonts"
)

// UpdateOptions captures user-tunable flags for an Update call.
type UpdateOptions struct {
	// Force, if true, re-downloads fonts already at the current release.
	Force bool

	// KeepArchive, if true, retains the downloaded zip under Deps.ArchivesDir.
	KeepArchive bool

	// SkipCacheRefresh, if true, suppresses the final fc-cache invocation.
	SkipCacheRefresh bool
}

// Update launches fonts.Update for the given tags (empty = all installed)
// in a goroutine, translating callback events into engine.Event sent on the
// returned channel. The channel closes at termination.
//
// Unlike Install, Update accepts a slice of tags and processes the batch in
// a single fonts.Update call — fonts handles enumeration of "all installed"
// when Names is empty.
func (e *Engine) Update(ctx context.Context, tags []string, opts UpdateOptions) OpHandle {
	opID := e.nextOpID()
	em := newEmitter(ctx)

	go func() {
		defer em.Close()
		em.Send(StartedEvent{OpID: opID, Kind: "update"})

		// Tracks per-tag failures already surfaced via OnEvent so the
		// post-call sweep of result.Failures doesn't double-emit.
		emittedFailures := map[string]struct{}{}

		params := fonts.UpdateParams{
			Names:        tags,
			FontDir:      e.deps.FontDir,
			StatePath:    e.deps.StatePath,
			CatalogPath:  e.deps.CatalogPath,
			ArchivesDir:  e.deps.ArchivesDir,
			GitHub:       e.deps.GitHub,
			AssetURLBase: e.deps.AssetURLBase,
			Refresher:    e.deps.FontCache,
		}
		fontsOpts := fonts.UpdateOptions{
			Force:            opts.Force,
			KeepArchive:      opts.KeepArchive,
			SkipCacheRefresh: opts.SkipCacheRefresh,
			OnProgress: func(font string, written, total int64) {
				em.Send(ProgressEvent{
					OpID:    opID,
					Target:  font,
					Stage:   "download",
					Written: written,
					Total:   total,
				})
			},
			OnEvent: func(fe fonts.Event) {
				if fe.Kind == fonts.EventInstallError {
					emittedFailures[fe.Font] = struct{}{}
				}
				translateUpdateEvent(opID, fe, em.Send)
			},
		}

		var result *fonts.UpdateResult
		err := retry(ctx, func() error {
			res, ferr := fonts.Update(ctx, params, fontsOpts)
			result = res
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
		// fonts.Update records per-tag failures (e.g. "not installed" for
		// named tags absent from the manifest) in result.Failures without
		// emitting EventInstallError. Surface them here, skipping any tag
		// already reported through the OnEvent path to avoid double-emits.
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
		Resolve: noopResolve,
	}
}

// translateUpdateEvent converts a fonts.Event into engine.Events for Update
// semantics. EventInstallSuccess/EventInstallSkipped are reused by fonts to
// signal per-font outcomes for Update too — we surface them as "updated" /
// "already fresh" rather than "installed".
func translateUpdateEvent(opID OpID, fe fonts.Event, send func(Event)) {
	switch fe.Kind {
	case fonts.EventDownloadStart:
		send(LogEvent{OpID: opID, Target: fe.Font, Level: LevelInfo, Message: "downloading"})
	case fonts.EventExtractStart:
		send(LogEvent{OpID: opID, Target: fe.Font, Level: LevelInfo, Message: "extracting"})
	case fonts.EventExtractDone:
		send(LogEvent{OpID: opID, Target: fe.Font, Level: LevelInfo, Message: "extracted"})
	case fonts.EventCacheRefresh:
		send(StartedEvent{OpID: opID, Kind: "fc-cache"})
	case fonts.EventInstallSuccess:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedSuccess, Detail: "updated"})
	case fonts.EventInstallSkipped:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedSkipped, Detail: "already fresh"})
	case fonts.EventInstallError:
		send(FailedEvent{OpID: opID, Target: fe.Font, Err: fe.Err})
	}
}
