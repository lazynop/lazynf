package doctor

import (
	"fmt"
	"strings"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/fonts"
	"github.com/lazynop/lazynf/internal/state"
)

// checkOrphans flags FontDir subdirectories that match catalog font names but
// are not tracked in the manifest — candidates for `lazynf import`. Skips
// silently with a WARN if the catalog is not available; doctor stays offline.
// m and cat are loaded once by Run; m may be a non-nil empty manifest, cat may
// be nil to indicate "not cached".
func checkOrphans(fontDir string, m *state.Manifest, cat *cache.Catalog) []Check {
	const section = SectionOrphans

	if cat == nil {
		return []Check{{
			Section:  section,
			Title:    "scan",
			Severity: SeverityWarn,
			Detail:   "catalog not cached, skipping scan",
			Hint:     "run `lazynf list` to populate the catalog first",
		}}
	}

	installed := map[string]state.InstalledFont{}
	if m != nil {
		installed = m.Installed
	}

	orphans, err := fonts.FindOrphans(fontDir, cat.Fonts, installed)
	if err != nil {
		return []Check{{
			Section:  section,
			Title:    "scan",
			Severity: SeverityWarn,
			Detail:   fmt.Sprintf("cannot read font dir: %s", err),
		}}
	}

	if len(orphans) == 0 {
		return []Check{{
			Section:  section,
			Title:    "scan",
			Severity: SeverityOK,
			Detail:   "none",
		}}
	}

	return []Check{{
		Section:  section,
		Title:    "scan",
		Severity: SeverityWarn,
		Detail:   fmt.Sprintf("%d orphan(s): %s", len(orphans), strings.Join(orphans, ", ")),
		Hint:     "run `lazynf import --all` or `lazynf import <name>`",
	}}
}
