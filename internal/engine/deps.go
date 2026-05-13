package engine

import (
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/github"
)

// Deps collects all dependencies injected into the engine.
// Paths are absolute; interfaces enable fakes in tests.
type Deps struct {
	// On-disk paths.
	FontDir     string
	StatePath   string
	CatalogPath string
	ArchivesDir string

	// GitHub configuration.
	GitHub       *github.Client
	AssetURLBase string // defaults to fonts.DefaultAssetURLBase if ""

	// fc-cache refresher; fontcache.Default() for the OS-aware default.
	FontCache fontcache.Refresher

	// Testable hooks.
	Now func() time.Time

	// CatalogLoader override for tests (default: cache.Load).
	CatalogLoader func(path string) (*cache.Catalog, error)
}
