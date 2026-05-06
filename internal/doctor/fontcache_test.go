package doctor

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckFcCache_NotApplicableNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("Linux: real fc-cache check tested elsewhere")
	}
	checks := checkFcCache()
	require.Len(t, checks, 1)
	assert.Equal(t, "fc-cache", checks[0].Section)
	assert.Equal(t, SeverityOK, checks[0].Severity)
	assert.Contains(t, checks[0].Detail, "not applicable")
}

func TestCheckFcCache_LinuxBehaviour(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("non-Linux: covered by TestCheckFcCache_NotApplicableNonLinux")
	}
	checks := checkFcCache()
	require.Len(t, checks, 1)
	c := checks[0]
	assert.Equal(t, "fc-cache", c.Section)
	// Severity is OK if the binary is in PATH on the CI/dev box,
	// else WARN. Both are valid; assert only that it's not FAIL
	// and that the detail string is non-empty.
	assert.NotEqual(t, SeverityFail, c.Severity)
	assert.NotEmpty(t, c.Detail)
}
