package fonts

import (
	"errors"
	"fmt"
	"os"

	"github.com/lazynop/lazynf/internal/state"
)

// Action is the install pipeline's decision after looking at on-disk + state.
type Action int

const (
	ActionInstall   Action = iota // dir does not exist; do a fresh install
	ActionReinstall               // overwrite (either for an update, or --force)
	ActionSkip                    // already at this release; do nothing
	ActionAbort                   // unexpected state, surface ErrConflict (FilesOnDisk)
	// ActionConflictImported signals that the manifest entry is the "imported"
	// sentinel and --force was not passed. The caller is expected to surface a
	// conflict event so the user can decide whether to overwrite.
	ActionConflictImported
)

// DetectConflict looks at the install dir and the manifest to decide what to do.
//
// Strict policy: if the dir exists but is NOT recorded in the manifest, refuse
// unless --force is passed (caller is expected to set `force=true`).
//
// Returned (action, err) pairs:
//   - (ActionInstall, nil)                       — fresh install, dir does not exist
//   - (ActionReinstall, nil)                     — different release, OR --force
//   - (ActionSkip, ErrAlreadyInstalled)          — same release, no force; caller logs and continues
//   - (ActionConflictImported, ErrAlreadyInstalled) — manifest entry is "imported", no force; caller asks user
//   - (ActionAbort, ErrConflict)                 — dir exists, not lazynf-managed, no force
func DetectConflict(m *state.Manifest, fontName, installDir, currentRelease string, force bool) (Action, error) {
	dirExists := pathExists(installDir)
	managed, isManaged := m.Installed[fontName]

	switch {
	case !dirExists:
		return ActionInstall, nil

	case isManaged && managed.IsImported() && !force:
		return ActionConflictImported, fmt.Errorf("%w: %s imported", ErrAlreadyInstalled, fontName)

	case isManaged && managed.Release == currentRelease && !force:
		return ActionSkip, fmt.Errorf("%w: %s at %s", ErrAlreadyInstalled, fontName, currentRelease)

	case isManaged && managed.Release == currentRelease && force:
		return ActionReinstall, nil

	case isManaged && managed.Release != currentRelease:
		return ActionReinstall, nil

	case !isManaged && force:
		return ActionReinstall, nil

	default: // !isManaged && !force
		return ActionAbort, fmt.Errorf("%w: %s exists at %s", ErrConflict, fontName, installDir)
	}
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return !errors.Is(err, os.ErrNotExist)
}
