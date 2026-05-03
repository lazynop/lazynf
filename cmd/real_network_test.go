package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lazynop/vellum/cmd"
)

// TestE2E_Real_Install_0xProto downloads a real (small) Nerd Font from the upstream repo
// and verifies the install pipeline runs end-to-end against live GitHub.
//
// Skipped unless VELLUM_E2E_REAL=1. Requires network access.
func TestE2E_Real_Install_0xProto(t *testing.T) {
	if os.Getenv("VELLUM_E2E_REAL") != "1" {
		t.Skip("set VELLUM_E2E_REAL=1 to run real-network tests")
	}

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	root := cmd.NewRoot("test")
	root.SetArgs([]string{"install", "0xProto", "--no-cache-refresh"})
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	if err := root.Execute(); err != nil {
		t.Fatalf("install failed: %s\n--- stderr ---\n%s", err, stderr.String())
	}

	t.Logf("stdout:\n%s", stdout.String())
}
