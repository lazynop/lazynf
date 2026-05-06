package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lazynop/lazynf/internal/doctor"
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
			params := doctor.Params{
				FontDir:     xdg.DefaultFontDir(),
				StatePath:   xdg.StateFile(),
				CatalogPath: xdg.CatalogFile(),
				ArchivesDir: xdg.ArchivesDir(),
				GitHub:      newGitHubClient(),
			}
			res, err := doctor.Run(params)
			if err != nil {
				return err
			}
			renderDoctorReport(v, res)
			if res.MaxSeverity() == doctor.SeverityFail {
				return errors.New("doctor: one or more checks failed")
			}
			return nil
		},
	}
}

// renderDoctorReport writes the doctor result to v.Stdout. On a TTY (and not
// quiet) it emits the grouped pretty layout; otherwise it emits one line per
// non-OK finding in a stable parseable shape.
func renderDoctorReport(v *ui.Verbosity, res *doctor.Result) {
	if v.ShouldShowProgress() {
		renderDoctorPretty(v, res)
		return
	}
	renderDoctorPlain(v, res)
}

func renderDoctorPretty(v *ui.Verbosity, res *doctor.Result) {
	fmt.Fprintln(v.Stdout, "lazynf doctor")
	prev := ""
	for _, c := range res.Checks {
		if c.Section != prev {
			fmt.Fprintln(v.Stdout)
			fmt.Fprintln(v.Stdout, c.Section)
			prev = c.Section
		}
		icon := iconForSeverity(c.Severity)
		line := fmt.Sprintf("  %s %s", icon, c.Title)
		if c.Detail != "" {
			line += "  " + ui.StyleDim.Render(c.Detail)
		}
		fmt.Fprintln(v.Stdout, line)
		if c.Hint != "" && c.Severity != doctor.SeverityOK {
			fmt.Fprintf(v.Stdout, "      %s %s\n", ui.StyleDim.Render("hint:"), c.Hint)
		}
	}

	_, warn, fail := res.Counts()
	fmt.Fprintln(v.Stdout)
	fmt.Fprintf(v.Stdout, "Summary: %d warnings, %d failures\n", warn, fail)
}

func renderDoctorPlain(v *ui.Verbosity, res *doctor.Result) {
	for _, c := range res.Checks {
		if c.Severity == doctor.SeverityOK {
			continue
		}
		tag := "WARN"
		if c.Severity == doctor.SeverityFail {
			tag = "FAIL"
		}
		section := doctor.SectionTag[c.Section]
		if section == "" {
			// Defensive: a future check could add a Section without a tag.
			section = strings.ToLower(c.Section)
		}
		fmt.Fprintf(v.Stdout, "%s %s %s\n", tag, section, c.Detail)
	}
}

func iconForSeverity(s doctor.Severity) string {
	switch s {
	case doctor.SeverityWarn:
		return ui.StyleWarn.Render("!")
	case doctor.SeverityFail:
		return ui.StyleFailure.Render("✗")
	default:
		return ui.StyleSuccess.Render("✓")
	}
}
