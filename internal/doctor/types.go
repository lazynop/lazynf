// Package doctor implements the read-only diagnostic command surface.
//
// Each check function returns a slice of Check; Run aggregates them in a
// fixed order into Result. No network calls; no automatic fixes — every
// non-OK check is paired with a Hint pointing to the existing user-facing
// command that resolves it.
package doctor

import (
	"github.com/lazynop/lazynf/internal/github"
)

// Severity ranks a Check from harmless to bug-blocking.
type Severity int

const (
	SeverityOK Severity = iota
	SeverityWarn
	SeverityFail
)

// Check is one finding in a doctor report.
type Check struct {
	Section  string // e.g. "XDG paths", "Manifest"
	Title    string // short label, e.g. "font dir", "schema version"
	Severity Severity
	Detail   string // human-readable detail, e.g. "/home/.../fonts" or "expected 8, found 6"
	Hint     string // optional remediation hint, e.g. "run `lazynf list`"
}

// Result is the full doctor output: an ordered slice of Check.
type Result struct {
	Checks []Check
}

// MaxSeverity returns the highest severity across all checks. Empty result
// returns SeverityOK.
func (r *Result) MaxSeverity() Severity {
	max := SeverityOK
	for _, c := range r.Checks {
		if c.Severity > max {
			max = c.Severity
		}
	}
	return max
}

// Counts returns the number of OK / WARN / FAIL checks.
func (r *Result) Counts() (ok, warn, fail int) {
	for _, c := range r.Checks {
		switch c.Severity {
		case SeverityOK:
			ok++
		case SeverityWarn:
			warn++
		case SeverityFail:
			fail++
		}
	}
	return
}

// Params bundles the I/O dependencies the checks need.
type Params struct {
	FontDir     string
	StatePath   string
	CatalogPath string
	ArchivesDir string
	GitHub      *github.Client
}
