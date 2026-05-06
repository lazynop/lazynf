package doctor

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/state"
)

// checkOrphans scans fontDir for subdirectories whose name matches a font in
// the catalog but that are not present in the manifest — candidates for
// `lazynf import`. Skips silently with a WARN if the catalog is not available.
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

	catalogSet := make(map[string]struct{}, len(cat.Fonts))
	for _, name := range cat.Fonts {
		catalogSet[name] = struct{}{}
	}

	entries, err := os.ReadDir(fontDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Check{{
				Section:  section,
				Title:    "scan",
				Severity: SeverityOK,
				Detail:   "none (font dir does not exist yet)",
			}}
		}
		return []Check{{
			Section:  section,
			Title:    "scan",
			Severity: SeverityWarn,
			Detail:   fmt.Sprintf("cannot read font dir: %s", err),
		}}
	}

	installed := map[string]state.InstalledFont{}
	if m != nil {
		installed = m.Installed
	}

	var orphans []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, inCatalog := catalogSet[name]; !inCatalog {
			continue
		}
		if _, inManifest := installed[name]; inManifest {
			continue
		}
		orphans = append(orphans, name)
	}

	if len(orphans) == 0 {
		return []Check{{
			Section:  section,
			Title:    "scan",
			Severity: SeverityOK,
			Detail:   "none",
		}}
	}

	sort.Strings(orphans)
	return []Check{{
		Section:  section,
		Title:    "scan",
		Severity: SeverityWarn,
		Detail:   fmt.Sprintf("%d orphan(s): %s", len(orphans), strings.Join(orphans, ", ")),
		Hint:     "run `lazynf import --all` or `lazynf import <name>`",
	}}
}
