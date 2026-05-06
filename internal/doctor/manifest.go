package doctor

import (
	"fmt"
	"os"

	"github.com/lazynop/lazynf/internal/state"
)

// checkManifest reports on a manifest already loaded by Run. Inputs:
//   - exists: state.json existed on disk (distinguishes "first run" from
//     "loaded successfully but empty").
//   - m: the parsed manifest (non-nil if loaded; may be empty).
//   - loadErr: parse failure (only meaningful when exists is true and m is nil).
func checkManifest(exists bool, m *state.Manifest, loadErr error) []Check {
	const section = SectionManifest

	if !exists {
		return []Check{{
			Section:  section,
			Title:    "state.json",
			Severity: SeverityOK,
			Detail:   "no manifest yet (first run)",
		}}
	}
	if loadErr != nil {
		return []Check{{
			Section:  section,
			Title:    "state.json",
			Severity: SeverityFail,
			Detail:   fmt.Sprintf("parse error: %s", loadErr),
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
