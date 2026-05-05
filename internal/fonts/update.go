package fonts

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"

	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/state"
)

// IsStale reports whether a font at the given recorded release should be
// refreshed. It returns true when release equals the imported sentinel (which
// carries no version information) or when release does not match currentTag.
//
// Fonts marked with the imported sentinel are always considered stale because
// their actual upstream tag is unknown — the next update fetches the current
// release and replaces the sentinel with the real tag.
func IsStale(release, currentTag string) bool {
	return release == state.ReleaseImported || release != currentTag
}

// Update re-downloads installed fonts whose recorded release differs from the
// current upstream tag (or that were imported with the "imported" sentinel).
// Internally it delegates to Install with Force=true for the stale subset, so
// the full install pipeline (download, extract, state update, fc-cache) is
// reused without duplication.
//
// Non-recoverable errors (catalog resolution, state load, state save) are
// returned as the function error. Per-font failures — including "not installed"
// for named fonts not in the manifest — are collected in UpdateResult.Failures.
func Update(ctx context.Context, p UpdateParams, opts UpdateOptions) (*UpdateResult, error) {
	if p.AssetURLBase == "" {
		p.AssetURLBase = DefaultAssetURLBase
	}
	if p.FontDir == "" {
		return nil, errors.New("update: FontDir is required")
	}
	if p.Refresher == nil {
		p.Refresher = fontcache.Default()
	}

	// Resolve catalog (may hit the network).
	cat, err := ResolveCatalog(p.GitHub, p.CatalogPath)
	if err != nil {
		return nil, err
	}

	manifest, err := state.Load(p.StatePath)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	res := &UpdateResult{Failures: map[string]error{}}

	// Build candidate list.
	var candidates []string
	if len(p.Names) > 0 {
		for _, name := range p.Names {
			if _, ok := manifest.Installed[name]; !ok {
				res.Failures[name] = fmt.Errorf("%s: not installed; use `lazynf install` to install it first", name)
				continue
			}
			candidates = append(candidates, name)
		}
	} else {
		// All installed fonts — sort for deterministic ordering.
		for name := range manifest.Installed {
			candidates = append(candidates, name)
		}
		sort.Strings(candidates)
	}

	// Split candidates into stale and already-fresh.
	var stale []string
	for _, name := range candidates {
		entry := manifest.Installed[name]
		if opts.Force || IsStale(entry.Release, cat.Release) {
			stale = append(stale, name)
		} else {
			res.AlreadyFresh = append(res.AlreadyFresh, name)
		}
	}

	if len(stale) == 0 {
		return res, nil
	}

	// Delegate to Install with Force=true so DetectConflict returns
	// ActionReinstall and the install dir is wiped + re-extracted.
	// Pass CatalogOverride so Install skips the redundant ResolveCatalog call.
	installRes, err := Install(ctx, InstallParams{
		Names:           stale,
		FontDir:         p.FontDir,
		StatePath:       p.StatePath,
		CatalogPath:     p.CatalogPath,
		ArchivesDir:     p.ArchivesDir,
		GitHub:          p.GitHub,
		AssetURLBase:    p.AssetURLBase,
		Refresher:       p.Refresher,
		CatalogOverride: cat,
	}, InstallOptions{
		Force:            true,
		KeepArchive:      opts.KeepArchive,
		SkipCacheRefresh: opts.SkipCacheRefresh,
		OnProgress:       opts.OnProgress,
		OnEvent:          opts.OnEvent,
	})
	if err != nil {
		return res, err
	}

	res.Updated = installRes.Successes
	maps.Copy(res.Failures, installRes.Failures)
	// installRes.Skipped should be empty when Force=true; ignore it.

	return res, nil
}
