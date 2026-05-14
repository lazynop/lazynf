package engine

import (
	"context"
	"errors"

	"github.com/lazynop/lazynf/internal/fonts"
)

// InstallOptions captures user-tunable flags for a single Install call.
// Maps 1:1 with fonts.InstallOptions except for OnEvent/OnProgress which
// are internal to the adapter.
type InstallOptions struct {
	// Force, if true, overwrites already-installed fonts and non-managed dirs.
	Force bool

	// Dest overrides Deps.FontDir for this single call. Empty uses the engine default.
	Dest string

	// KeepArchive, if true, retains the downloaded zip under Deps.ArchivesDir.
	KeepArchive bool

	// SkipCacheRefresh, if true, suppresses the final fc-cache invocation.
	SkipCacheRefresh bool
}

// Install launches fonts.Install for a single tag in a goroutine, translating
// callback events into engine.Event sent on the returned channel. The channel
// is closed at termination.
//
// For Plan 1 this does NOT emit ConflictEvent — the current semantics is
// "Force=true overwrites, Force=false yields InstallError". Conflict modals
// are added in Plan 2.
func (e *Engine) Install(ctx context.Context, tag string, opts InstallOptions) OpHandle {
	opID := e.nextOpID()
	em := newEmitter(ctx)

	go func() {
		defer em.Close()
		em.Send(StartedEvent{OpID: opID, Target: tag, Kind: "install"})

		fontDir := opts.Dest
		if fontDir == "" {
			fontDir = e.deps.FontDir
		}

		params := fonts.InstallParams{
			Names:        []string{tag},
			FontDir:      fontDir,
			StatePath:    e.deps.StatePath,
			CatalogPath:  e.deps.CatalogPath,
			ArchivesDir:  e.deps.ArchivesDir,
			GitHub:       e.deps.GitHub,
			AssetURLBase: e.deps.AssetURLBase,
			Refresher:    e.deps.FontCache,
		}
		fontsOpts := fonts.InstallOptions{
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
				translateInstallEvent(opID, fe, em.Send)
			},
		}

		err := retry(ctx, func() error {
			_, ferr := fonts.Install(ctx, params, fontsOpts)
			return ferr
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				em.Send(CanceledEvent{OpID: opID, Target: tag})
				return
			}
			em.Send(FailedEvent{OpID: opID, Target: tag, Err: err, Retriable: isRetriableNetErr(err)})
		}
		// Per-font Completed/Failed have already been emitted by translateInstallEvent.
	}()

	return OpHandle{
		Events:  em.Events(),
		Resolve: noopResolve,
	}
}

// translateInstallEvent converts a fonts.Event into zero or more engine.Events.
func translateInstallEvent(opID OpID, fe fonts.Event, send func(Event)) {
	switch fe.Kind {
	case fonts.EventDownloadStart:
		send(LogEvent{OpID: opID, Target: fe.Font, Level: LevelInfo, Message: "downloading"})
	case fonts.EventDownloadDone:
		// silent — next ExtractStart already signals progress
	case fonts.EventExtractStart:
		send(LogEvent{OpID: opID, Target: fe.Font, Level: LevelInfo, Message: "extracting"})
	case fonts.EventExtractDone:
		send(LogEvent{OpID: opID, Target: fe.Font, Level: LevelInfo, Message: "extracted"})
	case fonts.EventCacheRefresh:
		send(StartedEvent{OpID: opID, Target: "", Kind: "fc-cache"})
	case fonts.EventInstallSuccess:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedSuccess, Detail: "installed"})
	case fonts.EventInstallSkipped:
		send(CompletedEvent{OpID: opID, Target: fe.Font, Kind: CompletedSkipped, Detail: "already installed"})
	case fonts.EventInstallError:
		send(FailedEvent{OpID: opID, Target: fe.Font, Err: fe.Err})
	}
}
