// Package logpane keeps the in-memory log ring; file.go owns the on-disk
// failure log used for post-mortem inspection.
package logpane

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxLogBytes = 1 << 20 // 1 MiB

// FileLogger writes one line per call to <stateDir>/lazynf/tui.log. When the
// file would exceed 1 MiB, it is renamed to tui.log.0 (overwriting any
// previous rotation) and a fresh tui.log is opened.
type FileLogger struct {
	path string
	mu   sync.Mutex
}

// NewFileLogger returns a logger writing to stateDir/lazynf/tui.log.
// stateDir is usually $XDG_STATE_HOME (~/.local/state by default).
func NewFileLogger(stateDir string) *FileLogger {
	return &FileLogger{path: filepath.Join(stateDir, "lazynf", "tui.log")}
}

// Write appends a timestamped line, rotating first if the existing file
// would exceed maxLogBytes. Errors are returned but typically swallowed by
// the caller (logging failures must not bring the TUI down).
func (l *FileLogger) Write(msg string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	if info, err := os.Stat(l.path); err == nil && info.Size() > maxLogBytes {
		_ = os.Rename(l.path, l.path+".0") // best-effort rotation
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s  %s\n", time.Now().Format(time.RFC3339), msg)
	return err
}

// Path returns the on-disk log path for users (used by the help screen).
func (l *FileLogger) Path() string { return l.path }
