package doctor

import (
	"os"
	"path/filepath"
)

// isWritable returns true if a temp file can be created and removed in dir.
// Returns false if dir does not exist (does NOT check the parent).
func isWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".lazynf-doctor-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// checkPaths verifies the four XDG-derived directories lazynf reads/writes:
// font dir, state dir (parent of state.json), catalog dir (parent of
// catalog.json), and archives dir.
func checkPaths(p Params) []Check {
	type pathSpec struct {
		title string
		dir   string
	}
	specs := []pathSpec{
		{title: "font dir", dir: p.FontDir},
		{title: "state dir", dir: filepath.Dir(p.StatePath)},
		{title: "catalog dir", dir: filepath.Dir(p.CatalogPath)},
		{title: "archives dir", dir: p.ArchivesDir},
	}
	out := make([]Check, 0, len(specs))
	for _, s := range specs {
		out = append(out, classifyPath(s.title, s.dir))
	}
	return out
}

func classifyPath(title, dir string) Check {
	c := Check{Section: SectionPaths, Title: title, Detail: dir}
	if pathExists(dir) {
		if isWritable(dir) {
			c.Severity = SeverityOK
			return c
		}
		c.Severity = SeverityFail
		c.Hint = "fix permissions on the directory"
		return c
	}
	// Walk up the parent chain until we find an existing ancestor; if it is
	// writable, MkdirAll(dir) would succeed at first use.
	ancestor := filepath.Dir(dir)
	for !pathExists(ancestor) {
		next := filepath.Dir(ancestor)
		if next == ancestor {
			break
		}
		ancestor = next
	}
	if pathExists(ancestor) && isWritable(ancestor) {
		c.Severity = SeverityWarn
		c.Detail = dir + " (will be created on first use)"
		return c
	}
	c.Severity = SeverityFail
	c.Detail = dir + " (parent missing or not writable)"
	return c
}
