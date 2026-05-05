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

func newImportCmd() *cobra.Command {
	var (
		flagDest   string
		flagDetect bool
		flagForce  bool
		flagAll    bool
	)
	c := &cobra.Command{
		Use:   "import [<font>...]",
		Short: "Adopt fonts already on disk into lazynf's state manifest",
		Long: "Use this to register fonts that were installed by another tool " +
			"(getnf, manual download, package manager) so lazynf can later list, " +
			"update, or remove them.\n\n" +
			"With --all, scans the font directory and imports every subdirectory " +
			"whose name matches a Nerd Fonts catalog entry.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flagAll && len(args) == 0 {
				return errors.New("specify at least one font name or use --all")
			}
			if flagAll && len(args) > 0 {
				return errors.New("--all is mutually exclusive with explicit font names")
			}

			v := Verbosity()

			fontDir := flagDest
			if fontDir == "" {
				fontDir = xdg.DefaultFontDir()
			}

			gh := newGitHubClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			params := fonts.ImportParams{
				Names:        args,
				All:          flagAll,
				Detect:       flagDetect,
				Force:        flagForce,
				FontDir:      fontDir,
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				AssetURLBase: assetURLBase(),
				GitHub:       gh,
			}

			opts := fonts.ImportOptions{
				OnEvent: func(e fonts.Event) {
					switch e.Kind {
					case fonts.EventImportStart:
						v.Debugf("importing %s…", e.Font)
					case fonts.EventImportSuccess:
						v.Info("%s %s", ui.StyleSuccess.Render("✓"), e.Font)
					case fonts.EventImportSkipped:
						v.Info("%s %s (already in state, use --force to re-import)", ui.StyleDim.Render("•"), e.Font)
					case fonts.EventImportError:
						// Per-font errors are surfaced in the final summary below.
					}
				},
			}

			res, err := fonts.Import(context.Background(), params, opts)
			if err != nil {
				return err
			}

			// Summary
			if v.Level != ui.LevelQuiet || len(res.Failures) > 0 {
				summarizeImport(v, res)
			}
			if len(res.Failures) > 0 {
				return errors.New("one or more fonts failed to import")
			}
			return nil
		},
	}
	c.Flags().StringVar(&flagDest, "dest", "", "override font dir scanned (default: $XDG_DATA_HOME/fonts)")
	c.Flags().BoolVar(&flagDetect, "detect", false, "hash-compare with latest release to detect actual version (downloads ~10-15 MB per font)")
	c.Flags().BoolVar(&flagForce, "force", false, "re-import even if already in state")
	c.Flags().BoolVar(&flagAll, "all", false, "import all Nerd Fonts found in the font dir")
	return c
}

func summarizeImport(v *ui.Verbosity, res *fonts.ImportResult) {
	if len(res.Imported) > 0 {
		v.Info("%s imported: %s", ui.StyleSuccess.Render("✓"), strings.Join(res.Imported, ", "))
	}
	if len(res.Skipped) > 0 {
		v.Info("%s already in state: %s", ui.StyleDim.Render("•"), strings.Join(res.Skipped, ", "))
	}
	if len(res.Failures) > 0 {
		for name, err := range res.Failures {
			v.Errorf("%s %s: %s", ui.StyleFailure.Render("✗"), name, err.Error())
		}
	}
}
