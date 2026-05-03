// Package ui provides user-visible output helpers for the CLI commands:
// styling, progress bars, spinners, and verbosity-aware loggers.
//
// All UI rendering is on the cmd/ side; internal/fonts stays UI-agnostic
// and only fires callbacks.
package ui

import (
	"fmt"
	"io"
	"os"
)

// Level controls how much output is produced.
type Level int

const (
	LevelNormal  Level = iota // default
	LevelQuiet                // -q: errors only
	LevelVerbose              // -v: default + [debug] lines on stderr
)

// Verbosity holds the current level + I/O writers (Stdout/Stderr can be swapped in tests).
type Verbosity struct {
	Level  Level
	Stdout io.Writer
	Stderr io.Writer
}

// New returns a Verbosity with the given level wired to os.Stdout/os.Stderr.
func New(level Level) *Verbosity {
	return &Verbosity{Level: level, Stdout: os.Stdout, Stderr: os.Stderr}
}

// Info prints to stdout unless quiet.
func (v *Verbosity) Info(format string, args ...any) {
	if v.Level == LevelQuiet {
		return
	}
	fmt.Fprintf(v.Stdout, format+"\n", args...)
}

// Errorf prints to stderr always (errors are visible at every level).
func (v *Verbosity) Errorf(format string, args ...any) {
	fmt.Fprintf(v.Stderr, format+"\n", args...)
}

// Debugf prints `[debug]`-prefixed lines to stderr only at LevelVerbose.
func (v *Verbosity) Debugf(format string, args ...any) {
	if v.Level != LevelVerbose {
		return
	}
	fmt.Fprintf(v.Stderr, "[debug] "+format+"\n", args...)
}

// ShouldShowProgress returns true when progress bars/spinners make sense:
// not in quiet mode, and stdout is a TTY.
func (v *Verbosity) ShouldShowProgress() bool {
	if v.Level == LevelQuiet {
		return false
	}
	f, ok := v.Stdout.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
