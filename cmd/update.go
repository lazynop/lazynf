package cmd

import (
	"context"
	"errors"
	"strings"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		flagDest        string
		flagForce       bool
		flagKeepArchive bool
		flagNoCacheRefr bool
	)
	c := &cobra.Command{
		Use:   "update [<font>...]",
		Short: "Update one or more installed Nerd Fonts to the latest release",
		Long: `Updates fonts whose recorded release no longer matches the upstream catalog
(or that were imported with the 'imported' sentinel). With no arguments,
updates all installed fonts that are stale. With --force, refreshes even fonts
already at the latest release.`,
		ValidArgsFunction: completeFromManifest,
		RunE: func(_ *cobra.Command, args []string) error {
			v := Verbosity()

			fontDir := flagDest
			if fontDir == "" {
				fontDir = xdg.DefaultFontDir()
			}

			gh := newGitHubClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			eng := engine.New(engine.Deps{
				FontDir:      fontDir,
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       gh,
				AssetURLBase: assetURLBase(),
				FontCache:    refresher(),
			})

			showProgress := v.ShouldShowProgress()
			bars := map[string]*ui.ProgressTracker{}
			var spin *ui.Spinner

			var (
				updated      []string
				alreadyFresh []string
				failures     = map[string]error{}
			)

			opts := engine.UpdateOptions{
				Force:            flagForce,
				KeepArchive:      flagKeepArchive,
				SkipCacheRefresh: flagNoCacheRefr,
			}

			ctx := context.Background()
			handle := eng.Update(ctx, args, opts)
			for ev := range handle.Events {
				switch e := ev.(type) {
				case engine.LogEvent:
					if e.Message == "downloading" {
						if showProgress {
							b := ui.NewProgress("Downloading " + e.Target)
							b.Start()
							bars[e.Target] = b
						} else {
							v.Info("Downloading %s...", e.Target)
						}
					}
				case engine.ProgressEvent:
					if !showProgress {
						continue
					}
					b := bars[e.Target]
					if b == nil {
						continue
					}
					var pct float64
					if e.Total > 0 {
						pct = float64(e.Written) / float64(e.Total)
					}
					b.Update(pct, "")
				case engine.StartedEvent:
					if e.Kind == engine.KindFcCache {
						if showProgress {
							spin = ui.NewSpinner("Refreshing font cache")
							spin.Start()
						} else {
							v.Info("Refreshing font cache...")
						}
					}
				case engine.CompletedEvent:
					switch e.Kind {
					case engine.CompletedSuccess:
						if b := bars[e.Target]; b != nil {
							b.Finish()
							delete(bars, e.Target)
						} else {
							v.Info("%s %s", ui.StyleSuccess.Render("✓"), e.Target)
						}
						updated = append(updated, e.Target)
					case engine.CompletedSkipped:
						v.Info("%s %s (already installed)", ui.StyleDim.Render("•"), e.Target)
						alreadyFresh = append(alreadyFresh, e.Target)
					}
				case engine.FailedEvent:
					if b := bars[e.Target]; b != nil {
						b.Fail(e.Err.Error())
						delete(bars, e.Target)
					} else if e.Target == "" && e.Err != nil {
						// Soft error (e.g. fc-cache failure): warn on stderr.
						v.Errorf("%s %s", ui.StyleWarn.Render("!"), e.Err.Error())
					}
					if e.Target != "" && e.Err != nil {
						failures[e.Target] = e.Err
					}
				}
			}

			if spin != nil {
				spin.Stop(len(failures) == 0)
			}

			if v.Level != ui.LevelQuiet || len(failures) > 0 {
				summarizeUpdate(v, updated, alreadyFresh, failures)
			}

			if len(failures) > 0 {
				return errors.New("one or more fonts failed to update")
			}
			return nil
		},
	}
	c.Flags().StringVar(&flagDest, "dest", "", "override font install dir (default: $XDG_DATA_HOME/fonts)")
	c.Flags().BoolVar(&flagForce, "force", false, "refresh even fonts already at the latest release")
	c.Flags().BoolVar(&flagKeepArchive, "keep-archive", false, "keep downloaded zips in the archives cache")
	c.Flags().BoolVar(&flagNoCacheRefr, "no-cache-refresh", false, "skip the final fc-cache invocation")
	return c
}

func summarizeUpdate(v *ui.Verbosity, updated, alreadyFresh []string, failures map[string]error) {
	if len(updated) == 0 && len(failures) == 0 {
		v.Info("All installed fonts are up to date.")
		return
	}
	if len(updated) > 0 {
		v.Info("%s updated: %s", ui.StyleSuccess.Render("✓"), strings.Join(updated, ", "))
	}
	if len(alreadyFresh) > 0 {
		v.Info("%s already up to date: %s", ui.StyleDim.Render("•"), strings.Join(alreadyFresh, ", "))
	}
	if len(failures) > 0 {
		for name, err := range failures {
			v.Errorf("%s %s: %s", ui.StyleFailure.Render("✗"), name, err.Error())
		}
	}
}
