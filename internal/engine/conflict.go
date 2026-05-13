package engine

// ConflictKind classifies the conflict that the engine encountered while
// processing a target. The TUI uses this to render a confirm modal with
// applicable choices.
type ConflictKind int

const (
	// ConflictAlreadyImported: the target is currently in the manifest as
	// an "imported" entry (adopted from disk without version detection).
	ConflictAlreadyImported ConflictKind = iota

	// ConflictFilesOnDisk: the target directory exists on disk but is not
	// tracked by lazynf (no manifest entry).
	ConflictFilesOnDisk

	// ConflictVersionDowngrade: the operation would replace an installed
	// version with an older one.
	ConflictVersionDowngrade
)

// ConflictChoice is the user's resolution of a ConflictEvent, sent back via
// OpHandle.Resolve.
type ConflictChoice int

const (
	// ChoiceCancel: abort the operation for this target without changes.
	ChoiceCancel ConflictChoice = iota

	// ChoiceSkip: skip this target and proceed with the rest of the batch.
	ChoiceSkip

	// ChoiceForce: overwrite the existing state (re-install, re-import, etc.).
	ChoiceForce

	// ChoiceImportAs: keep the existing files but record them in the manifest
	// (only valid for ConflictFilesOnDisk).
	ChoiceImportAs
)
