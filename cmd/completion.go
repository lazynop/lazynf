package cmd

import (
	"github.com/lazynop/lazynf/internal/cache"
	"github.com/lazynop/lazynf/internal/fonts"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

// completeFromCatalog suggests every font name in the cached catalog. Returns
// no suggestions if the catalog is missing or unreadable: completion never
// triggers a network call.
func completeFromCatalog(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cat, err := cache.Load(xdg.CatalogFile())
	if err != nil || cat == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return cat.Fonts, cobra.ShellCompDirectiveNoFileComp
}

// completeFromManifest suggests fonts recorded in state.json. state.Load
// returns an empty (non-nil) manifest when the file is missing, so the loop
// simply yields no entries — equivalent to "no suggestions".
func completeFromManifest(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	m, err := state.Load(xdg.StateFile())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	out := make([]string, 0, len(m.Installed))
	for name := range m.Installed {
		out = append(out, name)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeOrphans suggests dirs in FontDir whose name is in the catalog
// but absent from the manifest. Delegates to fonts.FindOrphans, the same
// helper internal/doctor/checkOrphans uses, so the filter rule has one home.
func completeOrphans(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cat, err := cache.Load(xdg.CatalogFile())
	if err != nil || cat == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	m, err := state.Load(xdg.StateFile())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	out, err := fonts.FindOrphans(xdg.DefaultFontDir(), cat.Fonts, m.Installed)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
