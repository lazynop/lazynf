package fonts

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/lazynop/vellum/internal/archive"
	"github.com/lazynop/vellum/internal/github"
	"github.com/lazynop/vellum/internal/state"
)

// Import records fonts that already exist on disk into Vellum's state manifest.
// It does NOT write or modify any font files. No fc-cache invocation is needed
// because the OS font cache is already valid for files that were already present.
//
// Non-recoverable errors (catalog resolution, state load, state save) are returned
// as the function error. Per-font failures are recorded in ImportResult.Failures.
func Import(ctx context.Context, p ImportParams, opts ImportOptions) (*ImportResult, error) {
	if p.AssetURLBase == "" {
		p.AssetURLBase = DefaultAssetURLBase
	}
	if p.FontDir == "" {
		return nil, fmt.Errorf("import: FontDir is required")
	}

	cat, err := ResolveCatalog(p.GitHub, p.CatalogPath)
	if err != nil {
		return nil, err
	}

	manifest, err := state.Load(p.StatePath)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	// If --all, scan FontDir for subdirs whose name matches a catalog entry.
	names := p.Names
	if p.All && len(names) == 0 {
		names, err = scanForCatalogFonts(p.FontDir, cat.Fonts)
		if err != nil {
			return nil, fmt.Errorf("scan font dir: %w", err)
		}
	}

	res := &ImportResult{
		Failures: map[string]error{},
		Details:  map[string]string{},
	}

	for _, name := range names {
		emit(opts.OnEvent, Event{Font: name, Kind: EventImportStart})

		// Validate: name must be in catalog.
		if !slices.Contains(cat.Fonts, name) {
			suggestions := Suggest(cat.Fonts, name, 3)
			ferr := wrapFontNotFound(name, suggestions)
			res.Failures[name] = ferr
			emit(opts.OnEvent, Event{Font: name, Kind: EventImportError, Err: ferr})
			continue
		}

		installDir := filepath.Join(p.FontDir, name)

		// The font directory must already exist.
		if !pathExists(installDir) {
			ferr := fmt.Errorf("font dir not found: %s", installDir)
			res.Failures[name] = ferr
			emit(opts.OnEvent, Event{Font: name, Kind: EventImportError, Err: ferr})
			continue
		}

		// Skip if already in state and --force is not set.
		if _, alreadyManaged := manifest.Installed[name]; alreadyManaged && !p.Force {
			res.Skipped = append(res.Skipped, name)
			emit(opts.OnEvent, Event{Font: name, Kind: EventImportSkipped})
			continue
		}

		// Collect .ttf/.otf files from the install dir.
		localFiles, err := listFontFiles(installDir)
		if err != nil {
			res.Failures[name] = err
			emit(opts.OnEvent, Event{Font: name, Kind: EventImportError, Err: err})
			continue
		}

		// Determine which release to record.
		release := state.ReleaseImported
		if p.Detect {
			detected, detErr := detectRelease(ctx, name, installDir, localFiles, cat.Release, p)
			if detErr != nil {
				// Detect failure is non-fatal: fall back to sentinel.
				release = state.ReleaseImported
			} else {
				release = detected
			}
		}

		manifest.Installed[name] = state.InstalledFont{
			Release:     release,
			InstalledAt: time.Now().UTC(),
			Dir:         installDir,
			Files:       localFiles,
		}

		res.Imported = append(res.Imported, name)
		res.Details[name] = release
		emit(opts.OnEvent, Event{Font: name, Kind: EventImportSuccess})
	}

	if err := manifest.Save(p.StatePath); err != nil {
		return res, fmt.Errorf("save manifest: %w", err)
	}
	return res, nil
}

// scanForCatalogFonts returns the names of FontDir subdirectories whose names
// appear in the catalog font list.
func scanForCatalogFonts(fontDir string, catalogFonts []string) ([]string, error) {
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

// listFontFiles returns the basenames of all .ttf/.otf files in dir.
func listFontFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read font dir %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".ttf" || ext == ".otf" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// detectRelease downloads the latest release zip, extracts it, then compares
// SHA-256 hashes of every font file against the local files. If all hashes
// match, it returns the catalog release tag. Otherwise it returns ReleaseImported.
func detectRelease(
	ctx context.Context,
	name, installDir string,
	localFiles []string,
	catRelease string,
	p ImportParams,
) (string, error) {
	// Build map of local file hashes.
	localHashes, err := hashFiles(installDir, localFiles)
	if err != nil {
		return state.ReleaseImported, fmt.Errorf("hash local files: %w", err)
	}

	// Download the latest release zip.
	tmpDir, err := os.MkdirTemp("", "vellum-detect-*")
	if err != nil {
		return state.ReleaseImported, fmt.Errorf("create tempdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, name+".zip")
	url := fmt.Sprintf("%s/%s/%s.zip", p.AssetURLBase, catRelease, name)
	if err := github.DownloadAsset(url, zipPath, nil); err != nil {
		return state.ReleaseImported, fmt.Errorf("download %s: %w", name, err)
	}

	// Extract to a separate tempdir.
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return state.ReleaseImported, fmt.Errorf("create extract dir: %w", err)
	}
	extractedFiles, err := archive.ExtractFonts(zipPath, extractDir)
	if err != nil {
		return state.ReleaseImported, fmt.Errorf("extract %s: %w", name, err)
	}

	// Build map of upstream file hashes.
	upstreamHashes, err := hashFiles(extractDir, extractedFiles)
	if err != nil {
		return state.ReleaseImported, fmt.Errorf("hash upstream files: %w", err)
	}

	if hashMapsEqual(localHashes, upstreamHashes) {
		return catRelease, nil
	}
	return state.ReleaseImported, nil
}

// hashFiles computes SHA-256 of each file in dir by basename and returns a
// map from basename to hex digest.
func hashFiles(dir string, basenames []string) (map[string]string, error) {
	m := make(map[string]string, len(basenames))
	for _, name := range basenames {
		h, err := sha256File(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		m[name] = h
	}
	return m, nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// hashMapsEqual returns true when a and b have the same keys and values.
func hashMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
