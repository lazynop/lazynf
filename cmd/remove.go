package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	cterm "github.com/charmbracelet/x/term"
	"github.com/lazynop/lazynf/internal/fonts"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

// checkTTY reports whether stdin is connected to a terminal. Overridable in
// tests via assignment.
var checkTTY = func() bool {
	return cterm.IsTerminal(os.Stdin.Fd())
}

// stdinReader is the source of input for the confirmation prompt. Overridable
// in tests.
var stdinReader io.Reader = os.Stdin

// errAborted is returned when the user declines the confirmation prompt.
// Causes exit 1 via cmd.Exit() with a brief "aborted by user" message.
var errAborted = errors.New("aborted by user")

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
			if flagAll && !flagYes && !checkTTY() {
				return errors.New("--all requires --yes when stdin is not a terminal")
			}

			v := Verbosity()

			if flagAll {
				manifest, err := state.Load(xdg.StateFile())
				if err != nil {
					return fmt.Errorf("load manifest: %w", err)
				}
				if len(manifest.Installed) == 0 {
					v.Info("no fonts to remove")
					return nil
				}
				args = make([]string, 0, len(manifest.Installed))
				for name := range manifest.Installed {
					args = append(args, name)
				}
				sort.Strings(args)

				if !flagYes {
					var installed, imported int
					for _, name := range args {
						if manifest.Installed[name].Release == state.ReleaseImported {
							imported++
						} else {
							installed++
						}
					}
					if err := confirmRemoveAll(v, installed, imported, flagPurge); err != nil {
						return err
					}
				}
			}

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

// confirmRemoveAll prompts the user via stderr and reads from stdinReader.
// Returns nil to proceed, errAborted to cancel.
func confirmRemoveAll(_ *ui.Verbosity, installed, imported int, purge bool) error {
	total := installed + imported
	var msg string
	switch {
	case purge:
		msg = fmt.Sprintf(
			"About to remove %d font(s) from the manifest. ALL files will be deleted from disk, including %d imported font(s) adopted from elsewhere.",
			total, imported,
		)
	case imported > 0:
		msg = fmt.Sprintf(
			"About to remove %d font(s) from the manifest (%d installed will be deleted from disk; %d imported will be de-adopted, files left on disk).",
			total, installed, imported,
		)
	default:
		msg = fmt.Sprintf("About to remove %d font(s) from the manifest.", total)
	}
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprint(os.Stderr, "Continue? [y/N] ")

	reader := bufio.NewReader(stdinReader)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return errAborted
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "y" || line == "yes" {
		return nil
	}
	return errAborted
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
