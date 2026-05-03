package fonts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lazynop/vellum/internal/archive"
	"github.com/lazynop/vellum/internal/fontcache"
	"github.com/lazynop/vellum/internal/github"
	"github.com/lazynop/vellum/internal/state"
)

// InstallParams are dependencies and paths the install pipeline needs.
// Callers (cmd/install.go and the future TUI) construct these once per call.
type InstallParams struct {
	Names []string

	FontDir     string // base dir under which <name>/ subdirs are created
	StatePath   string // path to state.json
	CatalogPath string // path to catalog.json
	ArchivesDir string // dir for kept archives (only used if KeepArchive)

	GitHub       *github.Client
	AssetURLBase string // e.g. "https://github.com/ryanoasis/nerd-fonts/releases/download"
	Refresher    fontcache.Refresher
}

// DefaultAssetURLBase is the canonical Nerd Fonts release asset base URL.
const DefaultAssetURLBase = "https://github.com/ryanoasis/nerd-fonts/releases/download"

// Install runs the batch install pipeline. Best-effort: per-font failures are
// recorded in InstallResult.Failures and the next font is attempted.
//
// Returns an error only for non-recoverable failures BEFORE the per-font loop
// (catalog resolution, state load). Per-font failures are reported in the result.
func Install(ctx context.Context, p InstallParams, opts InstallOptions) (*InstallResult, error) {
	if p.AssetURLBase == "" {
		p.AssetURLBase = DefaultAssetURLBase
	}
	if p.FontDir == "" {
		return nil, errors.New("install: FontDir is required")
	}

	// Resolve catalog (may hit the network).
	cat, err := ResolveCatalog(p.GitHub, p.CatalogPath)
	if err != nil {
		return nil, err
	}

	manifest, err := state.Load(p.StatePath)
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	res := &InstallResult{Failures: map[string]error{}}
	any := false

	for _, name := range p.Names {
		if !contains(cat.Fonts, name) {
			suggestions := Suggest(cat.Fonts, name, 3)
			err := wrapFontNotFound(name, suggestions)
			res.Failures[name] = err
			emit(opts.OnEvent, Event{Font: name, Kind: EventInstallError, Err: err})
			continue
		}

		installDir := filepath.Join(p.FontDir, name)
		action, conflictErr := DetectConflict(manifest, name, installDir, cat.Release, opts.Force)
		switch action {
		case ActionSkip:
			res.Skipped = append(res.Skipped, name)
			emit(opts.OnEvent, Event{Font: name, Kind: EventInstallSkipped})
			continue
		case ActionAbort:
			res.Failures[name] = conflictErr
			emit(opts.OnEvent, Event{Font: name, Kind: EventInstallError, Err: conflictErr})
			continue
		}

		if action == ActionReinstall {
			if err := os.RemoveAll(installDir); err != nil {
				res.Failures[name] = fmt.Errorf("clean previous install dir: %w", err)
				emit(opts.OnEvent, Event{Font: name, Kind: EventInstallError, Err: err})
				continue
			}
		}

		files, err := installOne(ctx, name, cat.Release, installDir, p, opts)
		if err != nil {
			res.Failures[name] = err
			emit(opts.OnEvent, Event{Font: name, Kind: EventInstallError, Err: err})
			continue
		}

		manifest.Installed[name] = state.InstalledFont{
			Release:     cat.Release,
			InstalledAt: time.Now().UTC(),
			Dir:         installDir,
			Files:       files,
		}
		any = true
		res.Successes = append(res.Successes, name)
		emit(opts.OnEvent, Event{Font: name, Kind: EventInstallSuccess, Files: files})
	}

	// Persist state regardless of fc-cache outcome.
	if err := manifest.Save(p.StatePath); err != nil {
		return res, fmt.Errorf("save manifest: %w", err)
	}

	// Refresh the OS font cache once at the end if anything changed.
	if any && !opts.SkipCacheRefresh {
		emit(opts.OnEvent, Event{Kind: EventCacheRefresh})
		if err := p.Refresher.Refresh(ctx); err != nil {
			// Soft failure: surface as event, do NOT mark batch as failed.
			emit(opts.OnEvent, Event{Kind: EventInstallError, Err: err})
		}
	}
	return res, nil
}

func installOne(ctx context.Context, name, release, installDir string, p InstallParams, opts InstallOptions) ([]string, error) {
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("%w: %s", ErrPermission, installDir)
		}
		return nil, fmt.Errorf("mkdir %s: %w", installDir, err)
	}

	tmpDir, err := os.MkdirTemp("", "vellum-dl-*")
	if err != nil {
		return nil, fmt.Errorf("create tempdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, name+".zip")
	url := fmt.Sprintf("%s/%s/%s.zip", p.AssetURLBase, release, name)

	emit(opts.OnEvent, Event{Font: name, Kind: EventDownloadStart})
	if err := github.DownloadAsset(url, zipPath, func(w, t int64) {
		if opts.OnProgress != nil {
			opts.OnProgress(name, w, t)
		}
	}); err != nil {
		return nil, err
	}
	emit(opts.OnEvent, Event{Font: name, Kind: EventDownloadDone})

	emit(opts.OnEvent, Event{Font: name, Kind: EventExtractStart})
	files, err := archive.ExtractFonts(zipPath, installDir)
	if err != nil {
		// Cleanup partial install dir on extraction failure.
		_ = os.RemoveAll(installDir)
		return nil, err
	}
	sort.Strings(files)
	emit(opts.OnEvent, Event{Font: name, Kind: EventExtractDone, Files: files})

	if opts.KeepArchive {
		if err := os.MkdirAll(p.ArchivesDir, 0o755); err == nil {
			kept := filepath.Join(p.ArchivesDir, fmt.Sprintf("%s-%s.zip", name, release))
			if err := copyFile(zipPath, kept); err != nil {
				// Don't fail the install just because archive keeping failed.
				emit(opts.OnEvent, Event{Font: name, Kind: EventInstallError, Err: fmt.Errorf("keep archive: %w", err)})
			}
		}
	}

	return files, nil
}

func wrapFontNotFound(name string, suggestions []string) error {
	if len(suggestions) == 0 {
		return fmt.Errorf("%w: %s", ErrFontNotFound, name)
	}
	return fmt.Errorf("%w: %s (did you mean: %v?)", ErrFontNotFound, name, suggestions)
}

func contains(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

func emit(fn func(Event), e Event) {
	if fn != nil {
		fn(e)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
