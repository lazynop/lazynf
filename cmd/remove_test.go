package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// setTTY swaps checkTTY for the duration of the test.
func setTTY(t *testing.T, isTTY bool) {
	t.Helper()
	prev := checkTTY
	checkTTY = func() bool { return isTTY }
	t.Cleanup(func() { checkTTY = prev })
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
	setTTY(t, false)

	err := runRemove(t, []string{"--all"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stdin is not a terminal")
}

func TestRemoveCmd_AllEmptyManifest_NoOp(t *testing.T) {
	withXDG(t)
	// No seedManifest call — manifest does not exist.
	setTTY(t, false)

	err := runRemove(t, []string{"--all", "--yes"})
	require.NoError(t, err) // exit 0
}

// seedMixedManifest seeds a manifest with installed and imported entries.
// Each installed entry has a real on-disk Dir under t.TempDir() so file
// deletion in Remove succeeds (Files is empty so deleteFontFiles loops
// zero times and then os.Remove on an empty Dir succeeds). Imported
// entries point at /tmp paths that are not created — Remove de-adopts
// them without touching disk.
func seedMixedManifest(t *testing.T, installed, imported []string) {
	t.Helper()
	m := &state.Manifest{
		SchemaVersion: state.CurrentSchemaVersion,
		Installed:     map[string]state.InstalledFont{},
	}
	base := t.TempDir()
	for _, n := range installed {
		dir := filepath.Join(base, n)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		m.Installed[n] = state.InstalledFont{Release: "v3.4.0", Dir: dir}
	}
	for _, n := range imported {
		m.Installed[n] = state.InstalledFont{Release: state.ReleaseImported, Dir: "/tmp/" + n}
	}
	statePath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "lazynf", "state.json")
	require.NoError(t, m.Save(statePath))
}

func TestRemoveCmd_AllYes_RemovesEverything(t *testing.T) {
	withXDG(t)
	installFakeRefresher(t)
	seedMixedManifest(t,
		[]string{"FiraCode", "Hack", "JetBrainsMono"},
		[]string{"Mononoki", "Inconsolata"},
	)
	setTTY(t, false)

	err := runRemove(t, []string{"--all", "--yes"})
	require.NoError(t, err)

	statePath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "lazynf", "state.json")
	m, err := state.Load(statePath)
	require.NoError(t, err)
	assert.Empty(t, m.Installed, "manifest should be empty after --all --yes")
}

func setStdin(t *testing.T, input string) {
	t.Helper()
	prev := stdinReader
	stdinReader = strings.NewReader(input)
	t.Cleanup(func() { stdinReader = prev })
}

func TestRemoveCmd_AllPromptYes_Proceeds(t *testing.T) {
	withXDG(t)
	installFakeRefresher(t)
	seedManifest(t, []string{"FiraCode"})
	setTTY(t, true)
	setStdin(t, "y\n")

	err := runRemove(t, []string{"--all"})
	require.NoError(t, err)

	m, err := state.Load(filepath.Join(os.Getenv("XDG_DATA_HOME"), "lazynf", "state.json"))
	require.NoError(t, err)
	assert.Empty(t, m.Installed)
}

func TestRemoveCmd_AllPromptYesUppercase_Proceeds(t *testing.T) {
	withXDG(t)
	installFakeRefresher(t)
	seedManifest(t, []string{"FiraCode"})
	setTTY(t, true)
	setStdin(t, "Y\n")

	err := runRemove(t, []string{"--all"})
	require.NoError(t, err)
}

func TestRemoveCmd_AllPromptYesWord_Proceeds(t *testing.T) {
	withXDG(t)
	installFakeRefresher(t)
	seedManifest(t, []string{"FiraCode"})
	setTTY(t, true)
	setStdin(t, "yes\n")

	err := runRemove(t, []string{"--all"})
	require.NoError(t, err)
}

func TestRemoveCmd_AllPromptEmpty_Aborts(t *testing.T) {
	withXDG(t)
	seedManifest(t, []string{"FiraCode"})
	setTTY(t, true)
	setStdin(t, "\n")

	err := runRemove(t, []string{"--all"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errAborted), "expected errAborted, got %v", err)

	// Manifest must be intact.
	m, _ := state.Load(filepath.Join(os.Getenv("XDG_DATA_HOME"), "lazynf", "state.json"))
	assert.Len(t, m.Installed, 1)
}

func TestRemoveCmd_AllPromptNo_Aborts(t *testing.T) {
	withXDG(t)
	seedManifest(t, []string{"FiraCode"})
	setTTY(t, true)
	setStdin(t, "n\n")

	err := runRemove(t, []string{"--all"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errAborted))
}

func TestRemoveCmd_AllPromptGarbage_Aborts(t *testing.T) {
	withXDG(t)
	seedManifest(t, []string{"FiraCode"})
	setTTY(t, true)
	setStdin(t, "foo\n")

	err := runRemove(t, []string{"--all"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errAborted))
}

func TestRemoveCmd_AllPromptEOF_Aborts(t *testing.T) {
	withXDG(t)
	seedManifest(t, []string{"FiraCode"})
	setTTY(t, true)
	setStdin(t, "")

	err := runRemove(t, []string{"--all"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errAborted))
}

// Verify the prompt text mentions installed/imported counts.
func TestRemoveCmd_AllPromptText_MentionsCounts(t *testing.T) {
	withXDG(t)
	seedMixedManifest(t,
		[]string{"FiraCode", "Hack"}, // 2 installed
		[]string{"Mononoki"},         // 1 imported
	)
	setTTY(t, true)
	setStdin(t, "n\n")

	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = origStderr })

	go func() {
		_ = runRemove(t, []string{"--all"})
		_ = w.Close()
	}()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	assert.Contains(t, out, "3 font")
	assert.Contains(t, out, "2 installed")
	assert.Contains(t, out, "1 imported")
	assert.Contains(t, out, "[y/N]")
}
