package cmd

import (
	"sort"

	"github.com/lazynop/vellum/internal/fonts"
	"github.com/lazynop/vellum/internal/github"
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
				for _, n := range names {
					v.Info("%s  %s", n, ui.StyleDim.Render(m.Installed[n].Release))
				}
				return nil
			}

			gh := github.NewClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			cat, err := fonts.ResolveCatalog(gh, xdg.CatalogFile())
			if err != nil {
				return err
			}
			for _, n := range cat.Fonts {
				v.Info("%s", n)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&flagInstalled, "installed", false, "show only installed fonts")
	return c
}
