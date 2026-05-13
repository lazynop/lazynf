package cmd

import (
	"context"
	"fmt"
	"os"

	cterm "github.com/charmbracelet/x/term"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var flagInstalled bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List available or installed fonts",
		RunE: func(_ *cobra.Command, _ []string) error {
			v := Verbosity()

			gh := newGitHubClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			eng := engine.New(engine.Deps{
				FontDir:      xdg.DefaultFontDir(),
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       gh,
				AssetURLBase: assetURLBase(),
				FontCache:    refresher(),
			})

			infos, err := eng.List(context.Background())
			if err != nil {
				return err
			}

			if flagInstalled {
				installed := filterInstalled(infos)

				if v.ShouldShowProgress() {
					fmt.Fprintln(v.Stdout, ui.RenderInstalledTable(installed))
				} else {
					plain := ui.RenderInstalledPlain(installed)
					if plain != "" {
						fmt.Fprintln(v.Stdout, plain)
					}
				}
				return nil
			}

			if v.ShouldShowProgress() {
				termWidth := terminalWidth(os.Stdout)
				grid := ui.RenderCatalogGrid(infos, termWidth)
				if grid != "" {
					fmt.Fprintln(v.Stdout, grid)
				}
			} else {
				plain := ui.RenderCatalogPlain(infos)
				if plain != "" {
					fmt.Fprintln(v.Stdout, plain)
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&flagInstalled, "installed", false, "show only installed fonts")
	return c
}

// filterInstalled returns the subset of infos representing fonts present on
// disk (any status other than Available).
func filterInstalled(infos []engine.FontInfo) []engine.FontInfo {
	out := make([]engine.FontInfo, 0, len(infos))
	for _, fi := range infos {
		if fi.Status != engine.StatusAvailable {
			out = append(out, fi)
		}
	}
	return out
}

// terminalWidth returns the width of the given file's terminal, or a safe
// default of 80 if the file is not a TTY or the query fails.
func terminalWidth(f *os.File) int {
	w, _, err := cterm.GetSize(f.Fd())
	if err != nil || w <= 0 {
		return 80
	}
	return w
}
