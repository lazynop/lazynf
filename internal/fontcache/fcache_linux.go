//go:build linux

package fontcache

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// ErrFcCacheNotFound is returned when `fc-cache` is not present in PATH.
// Callers should treat this as a soft warning, not a failure of the install.
var ErrFcCacheNotFound = errors.New("fc-cache not found in PATH")

type linuxRefresher struct{}

func platformDefault() Refresher { return linuxRefresher{} }

func (linuxRefresher) Refresh(ctx context.Context) error {
	if _, err := exec.LookPath("fc-cache"); err != nil {
		return ErrFcCacheNotFound
	}
	cmd := exec.CommandContext(ctx, "fc-cache", "-f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fc-cache failed: %w (output: %s)", err, string(out))
	}
	return nil
}
