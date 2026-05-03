//go:build darwin

package fontcache

import "context"

type darwinRefresher struct{}

func platformDefault() Refresher { return darwinRefresher{} }

// Refresh is a no-op on macOS — CoreText scans ~/Library/Fonts on demand.
func (darwinRefresher) Refresh(_ context.Context) error { return nil }
