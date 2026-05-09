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

func newRemoveCmd() *cobra.Command {
	var (
		flagPurge          bool
		flagNoCacheRefresh bool
		flagAll            bool
		flagYes            bool
	)
	c := &cobra.Command{
		Use:   "remove <font>...",
		Short: "Remove installed fonts",
		Long: `Removes one or more fonts. By default, fonts installed via "lazynf install"
are deleted from disk and from the state manifest, while "imported" fonts are
only de-adopted from the manifest (their on-disk files are left intact). Use
--purge to also delete the on-disk directory of imported fonts.`,
		ValidArgsFunction: completeFromManifest,
		RunE: func(_ *cobra.Command, args []string) error {
			if flagAll && len(args) > 0 {
				return errors.New("--all is mutually exclusive with positional font names")
			}
			if !flagAll && len(args) == 0 {
				return errors.New("specify font names or --all")
			}
			if flagAll {
				return errors.New("--all not yet implemented")
			}

			v := Verbosity()

			showProgress := v.ShouldShowProgress()
			var spin *ui.Spinner

			params := fonts.RemoveParams{
				Names:     args,
				StatePath: xdg.StateFile(),
				Refresher: refresher(),
			}
			opts := fonts.RemoveOptions{
				Purge:            flagPurge,
				SkipCacheRefresh: flagNoCacheRefresh,
				OnEvent: func(e fonts.Event) {
					switch e.Kind {
					case fonts.EventRemoveSuccess:
						v.Info("%s %s", ui.StyleSuccess.Render("✓"), e.Font)
					case fonts.EventRemoveDeadopt:
						v.Info("%s %s (de-adopted)", ui.StyleDim.Render("•"), e.Font)
					case fonts.EventRemoveError:
						if e.Font == "" && e.Err != nil {
							// Soft error from fc-cache.
							v.Errorf("%s %s", ui.StyleWarn.Render("!"), e.Err.Error())
						}
						// Per-font errors are surfaced by summarizeRemove below.
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

			res, err := fonts.Remove(context.Background(), params, opts)
			if spin != nil {
				spin.Stop(err == nil)
			}
			if err != nil {
				return err
			}

			if v.Level != ui.LevelQuiet || len(res.Failures) > 0 {
				summarizeRemove(v, res)
			}

			if len(res.Failures) > 0 {
				return errors.New("one or more fonts failed to remove")
			}
			return nil
		},
	}
	c.Flags().BoolVar(&flagPurge, "purge", false, "also delete on-disk files for imported fonts")
	c.Flags().BoolVar(&flagNoCacheRefresh, "no-cache-refresh", false, "skip the final fc-cache invocation")
	c.Flags().BoolVar(&flagAll, "all", false, "remove every font in the manifest")
	c.Flags().BoolVarP(&flagYes, "yes", "y", false, "skip the confirmation prompt (required when stdin is not a terminal)")
	return c
}

func summarizeRemove(v *ui.Verbosity, res *fonts.RemoveResult) {
	if len(res.Removed) > 0 {
		v.Info("%s removed: %s", ui.StyleSuccess.Render("✓"), strings.Join(res.Removed, ", "))
	}
	if len(res.Deadopted) > 0 {
		v.Info("%s de-adopted (files left on disk): %s",
			ui.StyleDim.Render("•"), strings.Join(res.Deadopted, ", "))
	}
	if len(res.Failures) > 0 {
		for _, err := range res.Failures {
			v.Errorf("%s %s", ui.StyleFailure.Render("✗"), err.Error())
		}
	}
}
