package cmd

import (
	"fmt"
	"os"
	"sort"

	cterm "github.com/charmbracelet/x/term"
	"github.com/lazynop/vellum/internal/fonts"
	"github.com/lazynop/vellum/internal/state"
	"github.com/lazynop/vellum/internal/ui"
	"github.com/lazynop/vellum/internal/xdg"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var flagInstalled bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List available or installed fonts",
		RunE: func(_ *cobra.Command, _ []string) error {
			v := Verbosity()

			if flagInstalled {
				m, err := state.Load(xdg.StateFile())
				if err != nil {
					return err
				}
				names := make([]string, 0, len(m.Installed))
				for n := range m.Installed {
					names = append(names, n)
				}
				sort.Strings(names)

				if v.ShouldShowProgress() {
					fmt.Fprintln(v.Stdout, ui.RenderInstalledTable(names, m.Installed))
				} else {
					plain := ui.RenderInstalledPlain(names, m.Installed)
					if plain != "" {
						fmt.Fprintln(v.Stdout, plain)
					}
				}
				return nil
			}

			gh := newGitHubClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			cat, err := fonts.ResolveCatalog(gh, xdg.CatalogFile())
			if err != nil {
				return err
			}

			if v.ShouldShowProgress() {
				// TTY: resolve installed state for color markers.
				m, _ := state.Load(xdg.StateFile())
				installed := map[string]state.InstalledFont{}
				if m != nil {
					installed = m.Installed
				}

				termWidth := terminalWidth(os.Stdout)
				grid := ui.RenderCatalogGrid(cat.Fonts, installed, termWidth)
				if grid != "" {
					fmt.Fprintln(v.Stdout, grid)
				}
			} else {
				plain := ui.RenderCatalogPlain(cat.Fonts)
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

// terminalWidth returns the width of the given file's terminal, or a safe
// default of 80 if the file is not a TTY or the query fails.
func terminalWidth(f *os.File) int {
	w, _, err := cterm.GetSize(f.Fd())
	if err != nil || w <= 0 {
		return 80
	}
	return w
}
