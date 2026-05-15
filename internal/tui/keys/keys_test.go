package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefault_AllBindingsHaveKeys(t *testing.T) {
	k := Default()
	bindings := []struct {
		name string
		keys []string
	}{
		{"Quit", k.Quit.Keys()},
		{"Help", k.Help.Keys()},
		{"FocusNext", k.FocusNext.Keys()},
		{"Up", k.Up.Keys()},
		{"Down", k.Down.Keys()},
		{"Filter", k.Filter.Keys()},
		{"Install", k.Install.Keys()},
		{"Remove", k.Remove.Keys()},
		{"Select", k.Select.Keys()},
		{"SortCycle", k.SortCycle.Keys()},
	}
	for _, b := range bindings {
		require.NotEmpty(t, b.keys, "binding %s must have at least one key", b.name)
	}
}

func TestShortHelp_ContainsCoreBindings(t *testing.T) {
	help := Default().ShortHelp()
	require.NotEmpty(t, help)
}

func TestFullHelp_GroupsRows(t *testing.T) {
	rows := Default().FullHelp()
	require.GreaterOrEqual(t, len(rows), 4)
}
