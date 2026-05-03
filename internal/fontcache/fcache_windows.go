//go:build windows

package fontcache

import (
	"context"
	"errors"
)

// ErrUnsupported indicates Windows font registration is not yet implemented.
var ErrUnsupported = errors.New("vellum: Windows font cache refresh not implemented")

type windowsRefresher struct{}

func platformDefault() Refresher { return windowsRefresher{} }

func (windowsRefresher) Refresh(_ context.Context) error { return ErrUnsupported }
