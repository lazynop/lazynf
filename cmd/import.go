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
		ValidArgsFunction: completeOrphans,
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

			eng := engine.New(engine.Deps{
				FontDir:      fontDir,
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       gh,
				AssetURLBase: assetURLBase(),
				FontCache:    refresher(),
			})

			opts := engine.ImportOptions{
				All:    flagAll,
				Detect: flagDetect,
				Force:  flagForce,
			}

			var (
				imported []string
				skipped  []string
				failures = map[string]error{}
			)

			ctx := context.Background()
			handle := eng.Import(ctx, args, opts)
			for ev := range handle.Events {
				switch e := ev.(type) {
				case engine.LogEvent:
					if e.Message == "importing" {
						v.Debugf("importing %s…", e.Target)
					}
				case engine.CompletedEvent:
					switch e.Kind {
					case engine.CompletedSuccess:
						v.Info("%s %s", ui.StyleSuccess.Render("✓"), e.Target)
						imported = append(imported, e.Target)
					case engine.CompletedSkipped:
						v.Info("%s %s (already in state, use --force to re-import)", ui.StyleDim.Render("•"), e.Target)
						skipped = append(skipped, e.Target)
					}
				case engine.FailedEvent:
					if e.Target != "" && e.Err != nil {
						failures[e.Target] = e.Err
					}
				}
			}

			if v.Level != ui.LevelQuiet || len(failures) > 0 {
				summarizeImport(v, imported, skipped, failures)
			}
			if len(failures) > 0 {
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

func summarizeImport(v *ui.Verbosity, imported, skipped []string, failures map[string]error) {
	if len(imported) > 0 {
		v.Info("%s imported: %s", ui.StyleSuccess.Render("✓"), strings.Join(imported, ", "))
	}
	if len(skipped) > 0 {
		v.Info("%s already in state: %s", ui.StyleDim.Render("•"), strings.Join(skipped, ", "))
	}
	if len(failures) > 0 {
		for name, err := range failures {
			v.Errorf("%s %s: %s", ui.StyleFailure.Render("✗"), name, err.Error())
		}
	}
}
