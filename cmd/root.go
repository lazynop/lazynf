// Package cmd registers Vellum's Cobra commands and global flags.
//
// Commands here are thin: parse flags, build dependencies, call into
// internal/fonts, render results via internal/ui.
package cmd

import (
	"fmt"
	"os"

	"github.com/lazynop/vellum/internal/ui"
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
		Use:               "vellum",
		Short:             "Install Nerd Fonts from your terminal",
		Long:              "Vellum installs, lists, and searches Nerd Fonts. (TUI mode coming in a future release.)",
		Version:           version,
		SilenceErrors:     true, // we render errors ourselves
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flagQuiet && flagVerbose {
				return fmt.Errorf("--quiet and --verbose are mutually exclusive")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "errors only; no progress bars or spinners")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "extra diagnostic output on stderr")

	root.AddCommand(newInstallCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newCacheCmd())

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
