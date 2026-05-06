package doctor

import (
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
//
// Invariant: Result.Checks are appended grouped by Section in fixed order.
// The cmd-layer pretty renderer relies on contiguous-by-section emission to
// avoid printing a section header twice.
func Run(p Params) (*Result, error) {
	res := &Result{}

	// Hoist shared I/O. state.Load returns (empty, nil) for missing files;
	// the explicit pathExists check lets checkManifest distinguish "first run"
	// from "loaded successfully but empty" without a second os.Stat.
	manifestExists := pathExists(p.StatePath)
	var (
		m        *state.Manifest
		mLoadErr error
	)
	if manifestExists {
		m, mLoadErr = state.Load(p.StatePath)
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
