package engine

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/state"
)

// List returns the complete snapshot of known fonts: the union of catalog
// and manifest. Non-fatal errors (catalog missing, manifest missing) do NOT
// return an error: the function builds the best possible state from the
// available data. It returns an error only if either source exists and is
// corrupt (parse error).
func (e *Engine) List(ctx context.Context) ([]FontInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	loader := e.deps.CatalogLoader
	if loader == nil {
		loader = cache.Load
	}
	cat, catErr := loader(e.deps.CatalogPath)
	if catErr != nil && !os.IsNotExist(catErr) {
		return nil, catErr
	}
	// cat may be nil if the file does not exist — treated as "no catalog" below.

	manifest, err := state.Load(e.deps.StatePath)
	if err != nil {
		return nil, err
	}

	out := map[string]FontInfo{}

	// From catalog: every known tag, default status Available.
	if cat != nil {
		for _, name := range cat.Fonts {
			out[name] = FontInfo{
				Name:          name,
				Status:        StatusAvailable,
				LatestVersion: cat.Release,
			}
		}
	}

	// From manifest: replace/add with installed state.
	for name, entry := range manifest.Installed {
		fi := out[name] // zero value if not in catalog
		fi.Name = name
		fi.Dir = entry.Dir
		fi.InstalledAt = entry.InstalledAt
		fi.Files = append([]string(nil), entry.Files...)
		fi.Size = totalSize(entry.Dir, entry.Files)
		switch {
		case entry.IsImported():
			fi.Status = StatusImported
			fi.Version = ""
		case cat == nil:
			fi.Status = StatusUnknown
			fi.Version = entry.Release
		case cat.Release != "" && entry.Release != cat.Release:
			fi.Status = StatusStale
			fi.Version = entry.Release
			fi.LatestVersion = cat.Release
		default:
			fi.Status = StatusInstalled
			fi.Version = entry.Release
			fi.LatestVersion = cat.Release
		}
		out[name] = fi
	}

	result := make([]FontInfo, 0, len(out))
	for _, fi := range out {
		result = append(result, fi)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// totalSize stats every Files entry under Dir and sums the sizes. Errors are
// ignored (best-effort: List must not fail if a file was deleted outside
// lazynf).
func totalSize(dir string, files []string) int64 {
	var sum int64
	for _, f := range files {
		info, err := os.Stat(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		sum += info.Size()
	}
	return sum
}
