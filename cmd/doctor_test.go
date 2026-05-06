package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lazynop/lazynf/internal/doctor"
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
	res := &doctor.Result{Checks: []doctor.Check{
		// OK lines must be skipped on the plain output.
		{Section: doctor.SectionPaths, Title: "font dir", Severity: doctor.SeverityOK, Detail: "/x/fonts"},
		// Spec-listed tags: auth (not "github"), orphan (not "orphans").
		{Section: doctor.SectionGitHub, Title: "auth source", Severity: doctor.SeverityWarn,
			Detail: "anonymous (subject to lower rate limits)"},
		{Section: doctor.SectionOrphans, Title: "scan", Severity: doctor.SeverityWarn,
			Detail: "1 orphan(s): Hack"},
		// FAIL line.
		{Section: doctor.SectionManifest, Title: "state.json", Severity: doctor.SeverityFail,
			Detail: "parse error: invalid character"},
	}}

	var buf bytes.Buffer
	v := &ui.Verbosity{Level: ui.LevelNormal, Stdout: &buf, Stderr: &buf}
	renderDoctorPlain(v, res)

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
	res := &doctor.Result{Checks: []doctor.Check{
		{Section: "Made Up Section", Title: "x", Severity: doctor.SeverityWarn, Detail: "detail"},
	}}
	var buf bytes.Buffer
	v := &ui.Verbosity{Level: ui.LevelNormal, Stdout: &buf, Stderr: &buf}
	renderDoctorPlain(v, res)
	// Fallback lowercases the section name and replaces spaces with '-'
	// so the field-by-position contract holds for parsers like `awk '{print $2}'`.
	assert.Contains(t, buf.String(), "WARN made-up-section detail")
}
