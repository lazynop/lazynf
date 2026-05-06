package cmd

import (
	"context"
	"errors"
	"strings"

	"github.com/lazynop/lazynf/internal/fonts"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		flagDest        string
		flagKeepArchive bool
		flagForce       bool
		flagNoCacheRefr bool
	)
	c := &cobra.Command{
		Use:               "install <font>...",
		Short:             "Install one or more Nerd Fonts",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeFromCatalog,
		RunE: func(_ *cobra.Command, args []string) error {
			v := Verbosity()

			fontDir := flagDest
			if fontDir == "" {
				fontDir = xdg.DefaultFontDir()
			}

			gh := newGitHubClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			params := fonts.InstallParams{
				Names:        args,
				FontDir:      fontDir,
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       gh,
				AssetURLBase: assetURLBase(),
				Refresher:    refresher(),
			}

			showProgress := v.ShouldShowProgress()
			var bars = map[string]*ui.ProgressTracker{}
			var spin *ui.Spinner

			opts := fonts.InstallOptions{
				Force:            flagForce,
				KeepArchive:      flagKeepArchive,
				SkipCacheRefresh: flagNoCacheRefr,
				OnProgress: func(font string, written, total int64) {
					if !showProgress {
						return
					}
					b, ok := bars[font]
					if !ok || b == nil {
						return
					}
					var pct float64
					if total > 0 {
						pct = float64(written) / float64(total)
					}
					b.Update(pct, "")
				},
				OnEvent: func(e fonts.Event) {
					switch e.Kind {
					case fonts.EventDownloadStart:
						if showProgress {
							b := ui.NewProgress("Downloading " + e.Font)
							b.Start()
							bars[e.Font] = b
						} else {
							v.Info("Downloading %s...", e.Font)
						}
					case fonts.EventDownloadDone:
						// Bar stays open until install success/failure to avoid flicker
					case fonts.EventInstallSuccess:
						if b := bars[e.Font]; b != nil {
							b.Finish()
							delete(bars, e.Font)
						} else {
							v.Info("%s %s", ui.StyleSuccess.Render("✓"), e.Font)
						}
					case fonts.EventInstallSkipped:
						v.Info("%s %s (already installed)", ui.StyleDim.Render("•"), e.Font)
					case fonts.EventInstallError:
						if b := bars[e.Font]; b != nil {
							b.Fail(e.Err.Error())
							delete(bars, e.Font)
						} else if e.Font == "" && e.Err != nil {
							// Soft error (e.g. fc-cache failure): no per-font bar, warn on stderr.
							v.Errorf("%s %s", ui.StyleWarn.Render("!"), e.Err.Error())
						}
						// (per-font final summary line is printed below)
					case fonts.EventCacheRefresh:
						if showProgress {
							spin = ui.NewSpinner("Refreshing font cache")
							spin.Start()
						} else {
							v.Info("Refreshing font cache...")
						}
					}
				},
			}

			res, err := fonts.Install(context.Background(), params, opts)
			if spin != nil {
				spin.Stop(err == nil)
			}
			if err != nil {
				return err
			}

			// Summary
			if v.Level != ui.LevelQuiet || len(res.Failures) > 0 {
				summarize(v, res)
			}
			if len(res.Failures) > 0 {
				return errors.New("one or more fonts failed to install")
			}
			return nil
		},
	}
	c.Flags().StringVar(&flagDest, "dest", "", "override font install dir (default: $XDG_DATA_HOME/fonts)")
	c.Flags().BoolVar(&flagKeepArchive, "keep-archive", false, "keep downloaded zips in the archives cache")
	c.Flags().BoolVar(&flagForce, "force", false, "overwrite already-installed or non-lazynf-managed dirs")
	c.Flags().BoolVar(&flagNoCacheRefr, "no-cache-refresh", false, "skip the final fc-cache invocation")
	return c
}

func summarize(v *ui.Verbosity, res *fonts.InstallResult) {
	if len(res.Successes) > 0 {
		v.Info("%s installed: %s", ui.StyleSuccess.Render("✓"), strings.Join(res.Successes, ", "))
	}
	if len(res.Skipped) > 0 {
		v.Info("%s already installed: %s", ui.StyleDim.Render("•"), strings.Join(res.Skipped, ", "))
	}
	if len(res.Failures) > 0 {
		for name, err := range res.Failures {
			v.Errorf("%s %s: %s", ui.StyleFailure.Render("✗"), name, err.Error())
		}
	}
}
