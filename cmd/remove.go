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
	"github.com/lazynop/lazynf/internal/engine"
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
		Use:   "remove [<font>...]",
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
				var installed, imported int
				args = make([]string, 0, len(manifest.Installed))
				for name, entry := range manifest.Installed {
					args = append(args, name)
					if entry.IsImported() {
						imported++
					} else {
						installed++
					}
				}
				sort.Strings(args)

				if !flagYes {
					if err := confirmRemoveAll(installed, imported, flagPurge); err != nil {
						return err
					}
				}
			}

			showProgress := v.ShouldShowProgress()
			var spin *ui.Spinner

			eng := engine.New(engine.Deps{
				FontDir:      xdg.DefaultFontDir(),
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       newGitHubClient(),
				AssetURLBase: assetURLBase(),
				FontCache:    refresher(),
			})

			opts := engine.RemoveOptions{
				Purge:            flagPurge,
				SkipCacheRefresh: flagNoCacheRefresh,
			}

			var (
				removed   []string
				deadopted []string
				failures  = map[string]error{}
			)

			ctx := context.Background()
			handle := eng.Remove(ctx, args, opts)
			for ev := range handle.Events {
				switch e := ev.(type) {
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
						v.Info("%s %s", ui.StyleSuccess.Render("✓"), e.Target)
						removed = append(removed, e.Target)
					case engine.CompletedDeadopted:
						v.Info("%s %s (de-adopted)", ui.StyleDim.Render("•"), e.Target)
						deadopted = append(deadopted, e.Target)
					}
				case engine.FailedEvent:
					if e.Target == "" && e.Err != nil {
						// Soft error from fc-cache.
						v.Errorf("%s %s", ui.StyleWarn.Render("!"), e.Err.Error())
						continue
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
				summarizeRemove(v, removed, deadopted, failures)
			}

			if len(failures) > 0 {
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
func confirmRemoveAll(installed, imported int, purge bool) error {
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

func summarizeRemove(v *ui.Verbosity, removed, deadopted []string, failures map[string]error) {
	if len(removed) > 0 {
		v.Info("%s removed: %s", ui.StyleSuccess.Render("✓"), strings.Join(removed, ", "))
	}
	if len(deadopted) > 0 {
		v.Info("%s de-adopted (files left on disk): %s",
			ui.StyleDim.Render("•"), strings.Join(deadopted, ", "))
	}
	if len(failures) > 0 {
		for _, err := range failures {
			v.Errorf("%s %s", ui.StyleFailure.Render("✗"), err.Error())
		}
	}
}
