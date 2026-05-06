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

// checkOrphans scans FontDir for subdirectories whose name matches a font in
// the catalog but that are not present in the state manifest — candidates for
// `lazynf import`. Skips silently with a WARN if the catalog cache is not
// available; doctor stays offline.
func checkOrphans(fontDir, statePath, catalogPath string) []Check {
	const section = "Orphan directories"

	cat, err := cache.Load(catalogPath)
	if err != nil || cat == nil {
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

	m, err := state.Load(statePath)
	if err != nil {
		return []Check{{
			Section:  section,
			Title:    "scan",
			Severity: SeverityWarn,
			Detail:   fmt.Sprintf("manifest unreadable: %s", err),
		}}
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

	var orphans []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, inCatalog := catalogSet[name]; !inCatalog {
			continue
		}
		if _, inManifest := m.Installed[name]; inManifest {
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
