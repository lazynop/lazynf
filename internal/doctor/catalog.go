package doctor

import (
	"fmt"
	"time"

	"github.com/lazynop/lazynf/internal/cache"
)

// catalogStaleAfter is the age above which the cached catalog is considered
// stale and a refresh is recommended.
const catalogStaleAfter = 30 * 24 * time.Hour

// checkCatalog reports on a catalog already loaded by Run. Semantics of inputs:
//   - cat == nil && loadErr == nil: file absent (cache.Load convention).
//   - cat == nil && loadErr != nil: parse failure.
//   - cat != nil: loaded successfully.
func checkCatalog(cat *cache.Catalog, loadErr error) []Check {
	const section = SectionCatalog

	if loadErr != nil {
		return []Check{{
			Section:  section,
			Title:    "catalog.json",
			Severity: SeverityFail,
			Detail:   fmt.Sprintf("parse error: %s", loadErr),
		}}
	}
	if cat == nil {
		return []Check{{
			Section:  section,
			Title:    "catalog.json",
			Severity: SeverityWarn,
			Detail:   "not present",
			Hint:     "run `lazynf list` to populate",
		}}
	}

	age := time.Since(cat.CheckedAt)
	c := Check{
		Section: section,
		Title:   "catalog.json",
		Detail:  fmt.Sprintf("%d fonts (cached %s ago)", len(cat.Fonts), humanizeAge(age)),
	}
	if age >= catalogStaleAfter {
		c.Severity = SeverityWarn
		c.Hint = "run `lazynf list` to refresh"
	} else {
		c.Severity = SeverityOK
	}
	return []Check{c}
}

// humanizeAge formats a duration in a coarse, human-readable way:
// "<1m", "Nm", "Nh", "Nd". Uses integer-truncated values.
func humanizeAge(d time.Duration) string {
	if d < time.Minute {
		return "<1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
