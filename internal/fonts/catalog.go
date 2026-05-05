package fonts

import (
	"fmt"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/github"
)

// ResolveCatalog returns a fresh-or-cached catalog using getnf-style invalidation:
// always fetch the latest release tag, refresh the font list only when the tag
// differs from what's cached locally.
//
// Persists the result to catalogPath via atomic write.
func ResolveCatalog(gh *github.Client, catalogPath string) (*cache.Catalog, error) {
	tag, err := gh.LatestTag("ryanoasis", "nerd-fonts")
	if err != nil {
		return nil, fmt.Errorf("resolve catalog tag: %w", err)
	}

	// Treat a corrupt cache file the same as a missing one: re-fetch and
	// overwrite. The user gets a working tool instead of a hard error
	// requiring them to know about `lazynf cache clean`.
	cached, _ := cache.Load(catalogPath)
	if cached.IsFreshFor(tag) {
		return cached, nil
	}

	fonts, err := gh.PatchedFontsList()
	if err != nil {
		return nil, fmt.Errorf("resolve catalog list: %w", err)
	}

	updated := &cache.Catalog{
		SchemaVersion: cache.CurrentSchemaVersion,
		Release:       tag,
		CheckedAt:     time.Now().UTC(),
		Fonts:         fonts,
	}
	if err := updated.Save(catalogPath); err != nil {
		return nil, fmt.Errorf("save catalog cache: %w", err)
	}
	return updated, nil
}
