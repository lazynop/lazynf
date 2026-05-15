package engine

import (
	"context"
	"errors"
	"os"

	"github.com/lazynop/lazynf/internal/fonts"
)

// RefreshCatalog forces a network fetch of the Nerd Fonts catalog,
// ignoring the freshness of any local catalog file. Removes the existing
// catalog.json (if any) first so fonts.ResolveCatalog hits the network.
// Emits StartedEvent("catalog-fetch"), then CompletedEvent on success or
// FailedEvent on error.
func (e *Engine) RefreshCatalog(ctx context.Context) OpHandle {
	opID := e.nextOpID()
	em := newEmitter(ctx)

	go func() {
		defer em.Close()
		em.Send(StartedEvent{OpID: opID, Kind: KindCatalogFetch})

		// Remove the existing catalog so ResolveCatalog refetches.
		if err := os.Remove(e.deps.CatalogPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			em.Send(FailedEvent{OpID: opID, Err: err})
			return
		}

		err := retry(ctx, func() error {
			_, ferr := fonts.ResolveCatalog(e.deps.GitHub, e.deps.CatalogPath)
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
		em.Send(CompletedEvent{OpID: opID, Kind: CompletedSuccess, Detail: "catalog refreshed"})
	}()

	return OpHandle{Events: em.Events(), Resolve: noopResolve}
}
