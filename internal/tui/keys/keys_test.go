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

func TestShortHelp_SameSliceAcrossCalls(t *testing.T) {
	k := Default()
	a := k.ShortHelp()
	b := k.ShortHelp()
	require.NotEmpty(t, a)
	// Same backing array: proves no per-call allocation.
	require.Equal(t, &a[0], &b[0], "ShortHelp must return the cached slice")
}

func TestFullHelp_SameSliceAcrossCalls(t *testing.T) {
	k := Default()
	a := k.FullHelp()
	b := k.FullHelp()
	require.NotEmpty(t, a)
	require.NotEmpty(t, a[0])
	require.Equal(t, &a[0][0], &b[0][0])
}
