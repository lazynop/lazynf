package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/lazynf/internal/fontcache"
	"github.com/lazynop/lazynf/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installFakeRefresher swaps the package-level refresher with a no-op fake for
// the duration of the test. Required for any test that reaches fonts.Remove,
// otherwise fc-cache would run for real.
func installFakeRefresher(t *testing.T) {
	t.Helper()
	prev := testRefresher
	testRefresher = &fontcache.FakeRefresher{}
	t.Cleanup(func() { testRefresher = prev })
}

// runRemove builds the remove command in isolation and executes it with the
// given args. It returns the error from RunE (nil on success). Stdout/stderr
// are discarded for now; later tasks capture stderr to assert on prompt text.
func runRemove(t *testing.T, args []string) error {
	t.Helper()
	c := newRemoveCmd()
	c.SetArgs(args)
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	return c.Execute()
}

func TestRemoveCmd_NoArgsNoAll_Errors(t *testing.T) {
	withXDG(t)
	installFakeRefresher(t)
	err := runRemove(t, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify font names or --all")
}

func TestRemoveCmd_ArgsAndAll_Errors(t *testing.T) {
	withXDG(t)
	installFakeRefresher(t)
	seedManifest(t, []string{"FiraCode"})

	err := runRemove(t, []string{"FiraCode", "--all"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")

	// Manifest must not be touched.
	statePath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "lazynf", "state.json")
	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.Contains(t, m.Installed, "FiraCode", "manifest must not be touched")
}

// --yes without --all is silently accepted: it behaves identically to
// `remove <name>` because no prompt would have been shown anyway.
func TestRemoveCmd_YesWithoutAll_Accepted(t *testing.T) {
	withXDG(t)
	installFakeRefresher(t)
	seedManifest(t, []string{"FiraCode"})

	err := runRemove(t, []string{"FiraCode", "--yes"})
	require.NoError(t, err)

	m, _ := state.Load(filepath.Join(os.Getenv("XDG_DATA_HOME"), "lazynf", "state.json"))
	_, has := m.Installed["FiraCode"]
	assert.False(t, has, "FiraCode should be gone")
}

func TestRemoveCmd_AllNonTTYWithoutYes_Errors(t *testing.T) {
	withXDG(t)
	// Force non-TTY for this test.
	prev := checkTTY
	checkTTY = func() bool { return false }
	t.Cleanup(func() { checkTTY = prev })

	err := runRemove(t, []string{"--all"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stdin is not a terminal")
}
