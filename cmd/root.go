// Package cmd registers lazynf's Cobra commands and global flags.
//
// Commands here are thin: parse flags, build dependencies, call into
// internal/fonts, render results via internal/ui.
package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

// Globals populated by global flags. Read by sub-commands.
var (
	flagQuiet   bool
	flagVerbose bool
)

// NewRoot builds the root command tree.
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "lazynf",
		Short:         "Install Nerd Fonts from your terminal",
		Long:          "lazynf installs, lists, and searches Nerd Fonts. Run with no arguments on a TTY for the interactive TUI.",
		Version:       version,
		SilenceErrors: true, // we render errors ourselves
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flagQuiet && flagVerbose {
				return fmt.Errorf("--quiet and --verbose are mutually exclusive")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !isTerminal() {
				return cmd.Help()
			}
			gh := newGitHubClient()
			eng := engine.New(engine.Deps{
				FontDir:      xdg.DefaultFontDir(),
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       gh,
				AssetURLBase: assetURLBase(),
				FontCache:    refresher(),
			})
			return tui.Run(eng)
		},
	}

	root.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "errors only; no progress bars or spinners")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "extra diagnostic output on stderr")

	root.AddCommand(newInstallCmd())
	root.AddCommand(newImportCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newRemoveCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newCacheCmd())
	root.AddCommand(newDoctorCmd())

	return root
}

// Verbosity returns a *ui.Verbosity matching the global flags.
func Verbosity() *ui.Verbosity {
	switch {
	case flagQuiet:
		return ui.New(ui.LevelQuiet)
	case flagVerbose:
		return ui.New(ui.LevelVerbose)
	default:
		return ui.New(ui.LevelNormal)
	}
}

// Exit is the standard error-to-exit-code translator used by main.
func Exit(err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintln(os.Stderr, ui.StyleFailure.Render("error: ")+err.Error())
	return 1
}

// isTerminal reports whether stdout is a real terminal (TTY). The TUI is
// only launched when stdout is a TTY; piping or redirecting falls back to
// cobra's help output.
func isTerminal() bool {
	return term.IsTerminal(uintptr(os.Stdout.Fd()))
}
