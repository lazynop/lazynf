package cmd

import (
	"os"

	"github.com/lazynop/lazynf/internal/cache"
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
// returns an empty (but non-nil) manifest when the file is missing, so the
// loop simply yields no entries — equivalent to "no suggestions".
func completeFromManifest(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	m, err := state.Load(xdg.StateFile())
	if err != nil || m == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	out := make([]string, 0, len(m.Installed))
	for name := range m.Installed {
		out = append(out, name)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeOrphans suggests dirs in FontDir whose name is in the catalog
// but absent from the manifest — same set internal/doctor/orphans flags.
func completeOrphans(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cat, err := cache.Load(xdg.CatalogFile())
	if err != nil || cat == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	catalogSet := make(map[string]struct{}, len(cat.Fonts))
	for _, name := range cat.Fonts {
		catalogSet[name] = struct{}{}
	}
	m, err := state.Load(xdg.StateFile())
	if err != nil || m == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	entries, err := os.ReadDir(xdg.DefaultFontDir())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, inCatalog := catalogSet[name]; !inCatalog {
			continue
		}
		if _, inManifest := m.Installed[name]; inManifest {
			continue
		}
		out = append(out, name)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
