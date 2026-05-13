package cmd

import (
	"context"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/xdg"
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

			eng := engine.New(engine.Deps{
				FontDir:      xdg.DefaultFontDir(),
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       gh,
				AssetURLBase: assetURLBase(),
				FontCache:    refresher(),
			})

			infos, err := eng.Search(context.Background(), args[0])
			if err != nil {
				return err
			}
			for _, fi := range infos {
				v.Info("%s", fi.Name)
			}
			return nil
		},
	}
	return c
}
