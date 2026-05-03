// Package fontcache wraps platform-specific font cache refresh.
//
// On Linux, this invokes `fc-cache -f` to rebuild the fontconfig index.
// On macOS, no refresh is needed — CoreText discovers new files in
// ~/Library/Fonts automatically.
// On Windows, font registration is registry-based and not yet implemented.
package fontcache

import "context"

// Refresher rebuilds the platform's font index, if applicable.
type Refresher interface {
	Refresh(ctx context.Context) error
}

// Default returns the platform-appropriate Refresher for the current OS.
// This is what production code calls. Tests inject a fake instead.
func Default() Refresher {
	return platformDefault()
}

// FakeRefresher is a no-op Refresher with a recording flag, useful for tests.
type FakeRefresher struct {
	Called bool
	Err    error
}

func (f *FakeRefresher) Refresh(_ context.Context) error {
	f.Called = true
	return f.Err
}
