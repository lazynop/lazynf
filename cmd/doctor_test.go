package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lazynop/lazynf/internal/doctor"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderDoctorPlain is the contract used by scripts/CI/agents that pipe
// `lazynf doctor` output. The format is:
//
//	<TAG> <section-tag> <detail>
//
// where TAG is WARN/FAIL and section-tag is the canonical short label per
// doctor.SectionTag.
func TestRenderDoctorPlain_FormatAndSpecCompliance(t *testing.T) {
	sections := []engine.DoctorSectionEvent{
		// OK lines must be skipped on the plain output.
		{Section: doctor.SectionPaths, Title: "font dir", Status: engine.DoctorOK, Detail: "/x/fonts"},
		// Spec-listed tags: auth (not "github"), orphan (not "orphans").
		{Section: doctor.SectionGitHub, Title: "auth source", Status: engine.DoctorWarn,
			Detail: "anonymous (subject to lower rate limits)"},
		{Section: doctor.SectionOrphans, Title: "scan", Status: engine.DoctorWarn,
			Detail: "1 orphan(s): Hack"},
		// FAIL line.
		{Section: doctor.SectionManifest, Title: "state.json", Status: engine.DoctorFail,
			Detail: "parse error: invalid character"},
	}

	var buf bytes.Buffer
	v := &ui.Verbosity{Level: ui.LevelNormal, Stdout: &buf, Stderr: &buf}
	renderDoctorPlain(v, sections)

	out := buf.String()

	// OK lines are silent.
	assert.NotContains(t, out, "/x/fonts", "OK lines must not appear in plain output")

	// Spec-canonical short tags.
	assert.Contains(t, out, "WARN auth ")
	assert.Contains(t, out, "WARN orphan ")
	assert.NotContains(t, out, " github ", "tag must be 'auth', not 'github'")
	assert.NotContains(t, out, " orphans ", "tag must be 'orphan' (singular)")

	// FAIL prefix and detail flow through.
	assert.Contains(t, out, "FAIL manifest parse error: invalid character")

	// Three non-OK lines, three lines of output.
	require.Equal(t, 3, strings.Count(out, "\n"))
}

func TestRenderDoctorPlain_FallbackForUnknownSection(t *testing.T) {
	sections := []engine.DoctorSectionEvent{
		{Section: "Made Up Section", Title: "x", Status: engine.DoctorWarn, Detail: "detail"},
	}
	var buf bytes.Buffer
	v := &ui.Verbosity{Level: ui.LevelNormal, Stdout: &buf, Stderr: &buf}
	renderDoctorPlain(v, sections)
	// Fallback lowercases the section name and replaces spaces with '-'
	// so the field-by-position contract holds for parsers like `awk '{print $2}'`.
	assert.Contains(t, buf.String(), "WARN made-up-section detail")
}
