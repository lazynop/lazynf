package fonts

import (
	"sort"
	"strings"
)

// Search returns names containing the query as a case-insensitive substring,
// preserving alphabetical order. Empty query returns all names.
func Search(all []string, query string) []string {
	if query == "" {
		out := make([]string, len(all))
		copy(out, all)
		sort.Strings(out)
		return out
	}
	q := strings.ToLower(query)
	var out []string
	for _, n := range all {
		if strings.Contains(strings.ToLower(n), q) {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// Suggest returns up to `limit` candidate names from `all` that match `query`
// best by case-insensitive substring. Used to produce "did you mean ...?" hints
// for unknown font names. Order: substring matches first (alphabetical),
// then prefix matches if no substring hits.
func Suggest(all []string, query string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	matches := Search(all, query)
	if len(matches) >= limit {
		return matches[:limit]
	}
	if len(matches) > 0 {
		return matches
	}

	// Fall back to prefix-of-query matches (e.g., user typed "FiraCod" → "FiraCode")
	q := strings.ToLower(query)
	var prefixHits []string
	for _, n := range all {
		ln := strings.ToLower(n)
		if strings.HasPrefix(ln, q[:min(len(q), len(ln))]) {
			prefixHits = append(prefixHits, n)
		}
	}
	sort.Strings(prefixHits)
	if len(prefixHits) > limit {
		return prefixHits[:limit]
	}
	return prefixHits
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
