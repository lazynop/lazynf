package fonts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/state"
)

// Remove deletes one or more fonts. Behaviour depends on each font's record in
// the manifest:
//
//   - "installed" fonts (Release != state.ReleaseImported): files listed in
//     manifest.Installed[name].Files are removed from disk, the directory is
//     removed if it becomes empty, and the manifest entry is dropped.
//   - "imported" fonts (Release == state.ReleaseImported): by default only the
//     manifest entry is dropped (files are left on disk, since lazynf did not
//     create them). With opts.Purge=true, files are deleted as for installed
//     fonts — but only if Files is non-empty; otherwise this fails for that
//     font with a guidance error.
//
// fc-cache is invoked at most once per call, only if at least one file was
// actually deleted from disk. Per-font errors are collected in
// RemoveResult.Failures and do not abort the batch.
func Remove(ctx context.Context, p RemoveParams, opts RemoveOptions) (*RemoveResult, error) {
	if p.StatePath == "" {
		return nil, errors.New("remove: StatePath is required")
	}
	if p.Refresher == nil {
		p.Refresher = fontcache.Default()
	}

	manifest, err := state.Load(p.StatePath)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	res := &RemoveResult{Failures: map[string]error{}}

	for _, name := range p.Names {
		entry, ok := manifest.Installed[name]
		if !ok {
			err := fmt.Errorf("%s: not installed", name)
			res.Failures[name] = err
			emit(opts.OnEvent, Event{Font: name, Kind: EventRemoveError, Err: err})
			continue
		}

		isImported := entry.IsImported()

		if isImported && !opts.Purge {
			// De-adopt only: drop from manifest, leave files on disk.
			delete(manifest.Installed, name)
			res.Deadopted = append(res.Deadopted, name)
			emit(opts.OnEvent, Event{Font: name, Kind: EventRemoveDeadopt})
			continue
		}

		// Imported + --purge with no recorded files: refuse rather than
		// blindly RemoveAll a directory lazynf did not create.
		if isImported && opts.Purge && len(entry.Files) == 0 {
			err := fmt.Errorf("%s: no recorded files for imported font; run `lazynf import %s --detect` first, or delete the directory manually", name, name)
			res.Failures[name] = err
			emit(opts.OnEvent, Event{Font: name, Kind: EventRemoveError, Err: err})
			continue
		}

		// Delete files (installed font, or imported with --purge).
		if err := deleteFontFiles(entry); err != nil {
			res.Failures[name] = fmt.Errorf("remove %s: %w", name, err)
			emit(opts.OnEvent, Event{Font: name, Kind: EventRemoveError, Err: err})
			continue
		}
		delete(manifest.Installed, name)
		res.Removed = append(res.Removed, name)
		emit(opts.OnEvent, Event{Font: name, Kind: EventRemoveSuccess})
	}

	if len(res.Removed) > 0 || len(res.Deadopted) > 0 {
		if err := manifest.Save(p.StatePath); err != nil {
			return res, fmt.Errorf("save manifest: %w", err)
		}
	}

	if len(res.Removed) > 0 && !opts.SkipCacheRefresh {
		emit(opts.OnEvent, Event{Kind: EventCacheRefresh})
		if rerr := p.Refresher.Refresh(ctx); rerr != nil {
			emit(opts.OnEvent, Event{Kind: EventRemoveError, Err: rerr})
		}
	}

	return res, nil
}

// deleteFontFiles removes each file listed in entry.Files from entry.Dir.
// Missing files are ignored (idempotent). After deletion, the directory is
// removed if it became empty; otherwise it is left in place.
func deleteFontFiles(entry state.InstalledFont) error {
	for _, f := range entry.Files {
		path := filepath.Join(entry.Dir, f)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	_ = os.Remove(entry.Dir)
	return nil
}
