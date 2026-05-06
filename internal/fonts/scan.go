package fonts

import (
	"os"
	"slices"
	"sort"

	"github.com/lazynop/lazynf/internal/state"
)

// ScanCatalogDirs returns names of FontDir subdirectories whose names appear
// in the catalog. Returns nil (no error) if FontDir does not exist.
// Result is sorted alphabetically.
func ScanCatalogDirs(fontDir string, catalogFonts []string) ([]string, error) {
	entries, err := os.ReadDir(fontDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var matched []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if slices.Contains(catalogFonts, e.Name()) {
			matched = append(matched, e.Name())
		}
	}
	sort.Strings(matched)
	return matched, nil
}

// FindOrphans returns catalog-matching FontDir subdirectories that are NOT
// recorded in the manifest — candidates for `lazynf import`. Returns nil
// (no error) if FontDir does not exist.
func FindOrphans(fontDir string, catalogFonts []string, installed map[string]state.InstalledFont) ([]string, error) {
	matched, err := ScanCatalogDirs(fontDir, catalogFonts)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range matched {
		if _, inManifest := installed[name]; inManifest {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}
