package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_WideTerminal_SideBySide(t *testing.T) {
	l := Compute(120, 40, true)
	require.False(t, l.Vertical)
	require.Equal(t, 120, l.ListW+l.DetailW)
	require.Equal(t, 40, l.ListH+l.LogH+1)
}

func TestCompute_NarrowTerminal_Vertical(t *testing.T) {
	l := Compute(50, 30, true)
	require.True(t, l.Vertical)
	require.Equal(t, 50, l.ListW)
	require.Equal(t, 50, l.DetailW)
}

func TestCompute_HiddenLog_GrowsBody(t *testing.T) {
	l := Compute(120, 40, false)
	require.Equal(t, 0, l.LogH)
	require.Equal(t, 39, l.ListH)
}

func TestCompute_TinyTerminal_DoesNotPanic(t *testing.T) {
	_ = Compute(10, 5, true)
}
