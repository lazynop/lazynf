package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lazynop/lazynf/internal/doctor"
	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/ui"
	"github.com/lazynop/lazynf/internal/xdg"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose lazynf environment and state",
		Long: `Reports on font directories, fc-cache availability, GitHub auth source,
manifest integrity, catalog cache freshness, and orphan font directories.
No network calls. No automatic fixes — each issue points to the existing
command that resolves it.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			v := Verbosity()

			gh := newGitHubClient()
			v.Debugf("github auth source: %s", gh.AuthSource())

			eng := engine.New(engine.Deps{
				FontDir:      xdg.DefaultFontDir(),
				StatePath:    xdg.StateFile(),
				CatalogPath:  xdg.CatalogFile(),
				ArchivesDir:  xdg.ArchivesDir(),
				GitHub:       gh,
				AssetURLBase: assetURLBase(),
				FontCache:    refresher(),
			})

			var (
				sections []engine.DoctorSectionEvent
				opErr    error
			)
			handle := eng.RunDoctor(context.Background())
			for ev := range handle.Events {
				switch e := ev.(type) {
				case engine.DoctorSectionEvent:
					sections = append(sections, e)
				case engine.FailedEvent:
					if e.Err != nil {
						opErr = e.Err
					}
				}
			}
			if opErr != nil {
				return opErr
			}

			renderDoctorReport(v, sections)
			if hasFailure(sections) {
				return errors.New("doctor: one or more checks failed")
			}
			return nil
		},
	}
}

// hasFailure reports whether any section ended in DoctorFail.
func hasFailure(sections []engine.DoctorSectionEvent) bool {
	for _, s := range sections {
		if s.Status == engine.DoctorFail {
			return true
		}
	}
	return false
}

// countWarnFail returns the number of warning and failure sections.
func countWarnFail(sections []engine.DoctorSectionEvent) (warn, fail int) {
	for _, s := range sections {
		switch s.Status {
		case engine.DoctorWarn:
			warn++
		case engine.DoctorFail:
			fail++
		}
	}
	return
}

// renderDoctorReport writes the doctor result to v.Stdout. On a TTY (and not
// quiet) it emits the grouped pretty layout; otherwise it emits one line per
// non-OK finding in a stable parseable shape.
func renderDoctorReport(v *ui.Verbosity, sections []engine.DoctorSectionEvent) {
	if v.ShouldShowProgress() {
		renderDoctorPretty(v, sections)
		return
	}
	renderDoctorPlain(v, sections)
}

func renderDoctorPretty(v *ui.Verbosity, sections []engine.DoctorSectionEvent) {
	fmt.Fprintln(v.Stdout, "lazynf doctor")
	prev := ""
	for _, s := range sections {
		if s.Section != prev {
			fmt.Fprintln(v.Stdout)
			fmt.Fprintln(v.Stdout, s.Section)
			prev = s.Section
		}
		icon := iconForStatus(s.Status)
		line := fmt.Sprintf("  %s %s", icon, s.Title)
		if s.Detail != "" {
			line += "  " + ui.StyleDim.Render(s.Detail)
		}
		fmt.Fprintln(v.Stdout, line)
		if s.Hint != "" && s.Status != engine.DoctorOK {
			fmt.Fprintf(v.Stdout, "      %s %s\n", ui.StyleDim.Render("hint:"), s.Hint)
		}
	}

	warn, fail := countWarnFail(sections)
	fmt.Fprintln(v.Stdout)
	fmt.Fprintf(v.Stdout, "Summary: %d warnings, %d failures\n", warn, fail)
}

func renderDoctorPlain(v *ui.Verbosity, sections []engine.DoctorSectionEvent) {
	for _, s := range sections {
		if s.Status == engine.DoctorOK || s.Status == engine.DoctorSkip {
			continue
		}
		tag := "WARN"
		if s.Status == engine.DoctorFail {
			tag = "FAIL"
		}
		section := doctor.SectionTag[s.Section]
		if section == "" {
			// Defensive: a future check could add a Section without a tag.
			// Keep the field positional and space-free so awk-style parsing works.
			section = strings.ReplaceAll(strings.ToLower(s.Section), " ", "-")
		}
		fmt.Fprintf(v.Stdout, "%s %s %s\n", tag, section, s.Detail)
	}
}

func iconForStatus(s engine.DoctorStatus) string {
	switch s {
	case engine.DoctorWarn:
		return ui.StyleWarn.Render("!")
	case engine.DoctorFail:
		return ui.StyleFailure.Render("✗")
	default:
		return ui.StyleSuccess.Render("✓")
	}
}
