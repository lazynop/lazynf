package fonts

import (
	"time"

	"github.com/lazynop/vellum/internal/github"
)

// Event is a discrete state transition during install, surfaced via OnEvent
// callback so the UI layer can update its view without polling.
type EventKind int

const (
	EventDownloadStart  EventKind = iota // started downloading the zip
	EventDownloadDone                    // zip fully downloaded
	EventExtractStart                    // started extracting
	EventExtractDone                     // extracted N files
	EventCacheRefresh                    // running fc-cache (single event for the whole batch)
	EventInstallSuccess                  // font fully installed
	EventInstallSkipped                  // already installed at same release
	EventInstallError                    // failed (Err is non-nil)
	EventImportStart                     // started processing one font in import
	EventImportSuccess                   // font successfully recorded in state
	EventImportSkipped                   // font already in state and --force not set
	EventImportError                     // per-font import failure (Err is non-nil)
)

type Event struct {
	Font  string
	Kind  EventKind
	Files []string // for EventExtractDone
	Err   error    // for EventInstallError
}

// InstallOptions captures everything the caller can tune for one Install call.
type InstallOptions struct {
	// Force, if true, overwrites non-vellum-managed dirs and reinstalls already-current fonts.
	Force bool

	// KeepArchive, if true, moves the downloaded zip to the archives cache dir
	// instead of deleting it.
	KeepArchive bool

	// SkipCacheRefresh, if true, suppresses the final fc-cache invocation.
	SkipCacheRefresh bool

	// OnProgress is called frequently during downloads (per-read).
	// May be nil — the core does not depend on it.
	OnProgress func(font string, written, total int64)

	// OnEvent is called once per state transition. May be nil.
	OnEvent func(Event)
}

// InstallResult summarizes the outcome of a batch.
type InstallResult struct {
	Successes []string         // font names installed (or already-installed-skipped)
	Skipped   []string         // already-installed at same release
	Failures  map[string]error // font name -> error
}

// InstalledFontView is the per-font snapshot the catalog returns for `list --installed`.
type InstalledFontView struct {
	Name        string
	Release     string
	InstalledAt time.Time
}

// ImportParams are the dependencies and paths the import pipeline needs.
type ImportParams struct {
	Names []string // fonts to import; empty + All=true means scan FontDir

	All    bool // if true and Names is empty, scan FontDir for matching Nerd Font subdirs
	Detect bool // if true, hash-compare with latest release to detect actual version
	Force  bool // re-import even if already in state

	FontDir      string // base dir containing <name>/ subdirs
	StatePath    string // path to state.json
	CatalogPath  string // path to catalog.json
	AssetURLBase string // for --detect mode downloads

	GitHub *github.Client
}

// ImportOptions captures caller-tunable behaviour for a single Import call.
type ImportOptions struct {
	// OnEvent is called once per state transition. May be nil.
	OnEvent func(Event)
}

// ImportResult summarises the outcome of a batch import.
type ImportResult struct {
	Imported []string          // newly added or re-imported font names
	Skipped  []string          // already in state and --force not set
	Failures map[string]error  // font name → error
	Details  map[string]string // per-font detected release ("v3.4.0" or "imported")
}
