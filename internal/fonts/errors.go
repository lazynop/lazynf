// Package fonts is lazynf's UI-agnostic core: it orchestrates the GitHub client,
// archive extraction, state manifest, catalog cache, and font cache refresher
// to install Nerd Fonts. Cobra commands and (later) the TUI both call into here.
package fonts

import "errors"

// Sentinel errors that callers can match with errors.Is.
var (
	ErrFontNotFound     = errors.New("font not in catalog")
	ErrAlreadyInstalled = errors.New("font already installed at this release")
	ErrConflict         = errors.New("install dir exists and is not lazynf-managed")
	ErrPermission       = errors.New("filesystem permission denied")
)
