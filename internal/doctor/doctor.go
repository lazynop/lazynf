package doctor

import (
	"errors"
	"os"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/state"
)

// Run executes all diagnostic sections in fixed order and returns the
// aggregated Result. Never returns an error in the current design — every
// finding is reported through Check.Severity, not the function return.
//
// Run is the single source of truth for I/O against state.json and
// catalog.json: it loads each at most once and threads the parsed values into
// the checks that consume them.
func Run(p Params) (*Result, error) {
	res := &Result{}

	manifestExists := pathExists(p.StatePath)
	m, mLoadErr := state.Load(p.StatePath)
	// state.Load returns (empty, nil) for missing files; force m to nil when the
	// file did not exist so the manifest check can distinguish "first run" from
	// "loaded successfully but empty" without a second os.Stat.
	if !manifestExists {
		m = nil
	} else if mLoadErr != nil && errors.Is(mLoadErr, os.ErrNotExist) {
		// Defensive: state.Load should not surface ErrNotExist, but be safe.
		manifestExists = false
		mLoadErr = nil
	}

	cat, catLoadErr := cache.Load(p.CatalogPath)

	res.Checks = append(res.Checks, checkPaths(p)...)
	res.Checks = append(res.Checks, checkFcCache()...)
	res.Checks = append(res.Checks, checkGitHub(p.GitHub)...)
	res.Checks = append(res.Checks, checkManifest(manifestExists, m, mLoadErr)...)
	res.Checks = append(res.Checks, checkCatalog(cat, catLoadErr)...)
	res.Checks = append(res.Checks, checkOrphans(p.FontDir, m, cat)...)
	return res, nil
}
