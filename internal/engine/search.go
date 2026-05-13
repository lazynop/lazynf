package engine

import (
	"context"
	"strings"
)

// Search returns the FontInfos whose Name contains query (case-insensitive
// substring). An empty query returns the full List unchanged. Errors loading
// catalog/manifest propagate as-is.
func (e *Engine) Search(ctx context.Context, query string) ([]FontInfo, error) {
	all, err := e.List(ctx)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return all, nil
	}
	q := strings.ToLower(query)
	out := make([]FontInfo, 0, len(all))
	for _, fi := range all {
		if strings.Contains(strings.ToLower(fi.Name), q) {
			out = append(out, fi)
		}
	}
	return out, nil
}
