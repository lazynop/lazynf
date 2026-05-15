package confirm

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/lazynop/lazynf/internal/tui/keys"
	"github.com/lazynop/lazynf/internal/tui/messages"
	"github.com/stretchr/testify/require"
)

func TestYes_EmitsConfirmResultYes(t *testing.T) {
	m := New(keys.Default(), 42, "Remove FiraCode?", "Files will be deleted from disk.")
	_, cmd := m.Update(keyPress(t, "y"))
	require.NotNil(t, cmd)
	got, ok := cmd().(messages.ConfirmResultMsg)
	require.True(t, ok)
	require.Equal(t, int64(42), got.Token)
	require.Equal(t, messages.ChoiceYes, got.Choice)
}

func TestNo_EmitsConfirmResultNo(t *testing.T) {
	m := New(keys.Default(), 1, "Quit?", "")
	_, cmd := m.Update(keyPress(t, "n"))
	require.NotNil(t, cmd)
	require.Equal(t, messages.ChoiceNo, cmd().(messages.ConfirmResultMsg).Choice)
}

func TestEscape_EmitsCancel(t *testing.T) {
	m := New(keys.Default(), 1, "Quit?", "")
	_, cmd := m.Update(keyPress(t, "esc"))
	require.NotNil(t, cmd)
	require.Equal(t, messages.ChoiceCancel, cmd().(messages.ConfirmResultMsg).Choice)
}

func TestForce_EmitsForceWhenAllowed(t *testing.T) {
	m := New(keys.Default(), 1, "Title", "Body")
	m.AllowForce = true
	_, cmd := m.Update(keyPress(t, "f"))
	require.NotNil(t, cmd)
	require.Equal(t, messages.ChoiceForce, cmd().(messages.ConfirmResultMsg).Choice)
}

func TestForce_IgnoredWhenNotAllowed(t *testing.T) {
	m := New(keys.Default(), 1, "Title", "Body")
	m.AllowForce = false
	_, cmd := m.Update(keyPress(t, "f"))
	require.Nil(t, cmd, "f should be ignored when AllowForce is false")
}

func TestInit_ReturnsNil(t *testing.T) {
	require.Nil(t, New(keys.Default(), 1, "T", "B").Init())
}

func TestEnter_EmitsYes(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	_, cmd := m.Update(keyPress(t, "enter"))
	require.NotNil(t, cmd)
	require.Equal(t, messages.ChoiceYes, cmd().(messages.ConfirmResultMsg).Choice)
}

func TestNonKeyMsg_IsNoOp(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	_, cmd := m.Update("not a key")
	require.Nil(t, cmd)
}

func TestUnboundKey_NoCmd(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	_, cmd := m.Update(keyPress(t, "z"))
	require.Nil(t, cmd)
}

func TestAdopt_EmitsAdoptWhenAllowed(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	m.AllowAdopt = true
	_, cmd := m.Update(keyPress(t, "a"))
	require.NotNil(t, cmd)
	require.Equal(t, messages.ChoiceAdopt, cmd().(messages.ConfirmResultMsg).Choice)
}

func TestAdopt_IgnoredWhenNotAllowed(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	_, cmd := m.Update(keyPress(t, "a"))
	require.Nil(t, cmd)
}

func TestView_ZeroDimensions_UsesDefaults(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	require.NotEmpty(t, m.View().Content)
}

func TestView_AllowForceOnly_HintShowsForce(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	m.AllowForce = true
	m.Width, m.Height = 80, 24
	require.Contains(t, m.View().Content, "force")
}

func TestView_AllowAdoptOnly_HintShowsAdopt(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	m.AllowAdopt = true
	m.Width, m.Height = 80, 24
	require.Contains(t, m.View().Content, "adopt")
}

func TestView_ForceAndAdopt_HintShowsBoth(t *testing.T) {
	m := New(keys.Default(), 1, "T", "B")
	m.AllowForce = true
	m.AllowAdopt = true
	m.Width, m.Height = 80, 24
	s := m.View().Content
	require.Contains(t, s, "force")
	require.Contains(t, s, "adopt")
}

func TestRender_ShowsTitleAndBody(t *testing.T) {
	m := New(keys.Default(), 1, "TitleXyz", "BodyZyx")
	m.Width, m.Height = 80, 24
	s := m.View().Content
	require.Contains(t, s, "TitleXyz")
	require.Contains(t, s, "BodyZyx")
}

// keyPress constructs a tea.KeyPressMsg for the given key string. Special keys
// ("esc", "enter") are encoded via Code only; printable keys carry both Code
// and Text so that Key.String() returns the literal Text value used by the
// bindings in keys.Default().
func keyPress(t *testing.T, k string) tea.KeyPressMsg {
	t.Helper()
	switch k {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	default:
		return tea.KeyPressMsg{Code: rune(k[0]), Text: k}
	}
}
