package doctor

import (
	"os/exec"
	"testing"

	"github.com/lazynop/lazynf/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckGitHub_AuthEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "fake-token")
	c := github.NewClient()

	checks := checkGitHub(c)
	require.Len(t, checks, 1)
	assert.Equal(t, "GitHub auth", checks[0].Section)
	assert.Equal(t, SeverityOK, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "GITHUB_TOKEN")
}

func TestCheckGitHub_AuthNone(t *testing.T) {
	// Make sure no env-based or gh-based token leaks in.
	t.Setenv("GITHUB_TOKEN", "")
	if _, err := exec.LookPath("gh"); err == nil {
		t.Skip("gh CLI present; cannot reliably exercise the AuthNone branch here")
	}
	c := github.NewClient()

	checks := checkGitHub(c)
	require.Len(t, checks, 1)
	assert.Equal(t, "GitHub auth", checks[0].Section)
	assert.Equal(t, SeverityWarn, checks[0].Severity)
	assert.Contains(t, checks[0].Hint, "GITHUB_TOKEN")
}
