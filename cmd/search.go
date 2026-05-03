package cmd

import (
	"github.com/lazynop/vellum/internal/fonts"
	"github.com/lazynop/vellum/internal/xdg"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the catalog by case-insensitive substring",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			v := Verbosity()
			gh := newGitHubClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			cat, err := fonts.ResolveCatalog(gh, xdg.CatalogFile())
			if err != nil {
				return err
			}
			for _, n := range fonts.Search(cat.Fonts, args[0]) {
				v.Info("%s", n)
			}
			return nil
		},
	}
	return c
}
