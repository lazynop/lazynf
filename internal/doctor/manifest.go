package doctor

import (
	"errors"
	"fmt"
	"os"

	"github.com/lazynop/lazynf/internal/state"
)

// checkManifest validates the state.json: parseability, schema version, and
// for each entry that the recorded Dir exists on disk and the file count
// matches len(Files).
func checkManifest(statePath string) []Check {
	const section = "Manifest"

	// Distinguish "file does not exist" from other read errors. state.Load
	// returns a fresh empty manifest on ErrNotExist, hiding the distinction —
	// we want to report "no manifest yet (first run)" as OK, not as "0 fonts".
	if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
		return []Check{{
			Section:  section,
			Title:    "state.json",
			Severity: SeverityOK,
			Detail:   "no manifest yet (first run)",
		}}
	}

	m, err := state.Load(statePath)
	if err != nil {
		return []Check{{
			Section:  section,
			Title:    "state.json",
			Severity: SeverityFail,
			Detail:   fmt.Sprintf("parse error: %s", err),
		}}
	}

	if m.SchemaVersion > state.CurrentSchemaVersion {
		return []Check{{
			Section:  section,
			Title:    "schema version",
			Severity: SeverityFail,
			Detail: fmt.Sprintf(
				"manifest schema v%d is newer than supported (v%d)",
				m.SchemaVersion, state.CurrentSchemaVersion),
			Hint: "upgrade lazynf to a version that understands this schema",
		}}
	}

	out := []Check{{
		Section:  section,
		Title:    "schema version",
		Severity: SeverityOK,
		Detail:   fmt.Sprintf("v%d", m.SchemaVersion),
	}}

	for name, entry := range m.Installed {
		if !pathExists(entry.Dir) {
			out = append(out, Check{
				Section:  section,
				Title:    name,
				Severity: SeverityWarn,
				Detail:   fmt.Sprintf("%s: dir missing on disk (%s)", name, entry.Dir),
				Hint:     fmt.Sprintf("run `lazynf install %s` to recover", name),
			})
			continue
		}
		ents, err := os.ReadDir(entry.Dir)
		if err != nil {
			out = append(out, Check{
				Section:  section,
				Title:    name,
				Severity: SeverityWarn,
				Detail:   fmt.Sprintf("%s: cannot read dir (%s)", name, err),
			})
			continue
		}
		var fileCount int
		for _, e := range ents {
			if !e.IsDir() {
				fileCount++
			}
		}
		if fileCount != len(entry.Files) {
			out = append(out, Check{
				Section:  section,
				Title:    name,
				Severity: SeverityWarn,
				Detail: fmt.Sprintf(
					"%s: expected %d files, found %d on disk",
					name, len(entry.Files), fileCount),
				Hint: fmt.Sprintf("run `lazynf update %s` to refresh", name),
			})
		}
	}

	// Only the schema-version OK is in `out` if no per-entry issue was added.
	if len(out) == 1 {
		out = append(out, Check{
			Section:  section,
			Title:    "entries",
			Severity: SeverityOK,
			Detail:   fmt.Sprintf("%d fonts installed, all on disk", len(m.Installed)),
		})
	}
	return out
}
