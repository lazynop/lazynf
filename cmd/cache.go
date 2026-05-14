package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cache",
		Short: "Manage lazynf's catalog cache",
	}
	c.AddCommand(newCacheRefreshCmd())
	c.AddCommand(newCacheCleanCmd())
	return c
}

func newCacheRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Force a fresh fetch of the Nerd Fonts catalog",
		Long: `Removes the local catalog cache and re-fetches the latest release tag and
asset list from GitHub. Useful when a new Nerd Fonts release has just shipped
and you want to pick it up without waiting for the normal cache TTL.`,
		Args: cobra.NoArgs,
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

			var opErr error
			handle := eng.RefreshCatalog(context.Background())
			for ev := range handle.Events {
				switch e := ev.(type) {
				case engine.StartedEvent:
					if e.Kind == "catalog-fetch" {
						v.Info("Refreshing catalog...")
					}
				case engine.CompletedEvent:
					if e.Kind == engine.CompletedSuccess {
						v.Info("%s catalog refreshed", ui.StyleSuccess.Render("✓"))
					}
				case engine.FailedEvent:
					if e.Err != nil {
						opErr = e.Err
					}
				}
			}
			return opErr
		},
	}
}

func newCacheCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Clear the catalog cache and any kept archives",
		RunE: func(_ *cobra.Command, _ []string) error {
			v := Verbosity()
			catalog := xdg.CatalogFile()
			archives := xdg.ArchivesDir()

			catRemoved, err := removeIfPresent(catalog)
			if err != nil {
				return err
			}
			arcRemoved, err := removeAllIfPresent(archives)
			if err != nil {
				return err
			}

			if !catRemoved && !arcRemoved {
				v.Info("Cache already clean.")
				return nil
			}
			parts := ""
			if catRemoved {
				parts += "catalog"
			}
			if arcRemoved {
				if parts != "" {
					parts += " and "
				}
				parts += "archives"
			}
			v.Info("%s Cache cleared (%s removed).", ui.StyleSuccess.Render("✓"), parts)
			return nil
		},
	}
}

// removeIfPresent removes a single file. Returns (removed?, err).
func removeIfPresent(p string) (bool, error) {
	err := os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// removeAllIfPresent removes a directory recursively. Returns (removed?, err).
func removeAllIfPresent(p string) (bool, error) {
	_, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := os.RemoveAll(p); err != nil {
		return false, err
	}
	return true, nil
}
