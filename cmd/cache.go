package cmd

import (
	"errors"
	"os"

	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cache",
		Short: "Manage lazynf's catalog cache",
	}
	c.AddCommand(newCacheCleanCmd())
	return c
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
