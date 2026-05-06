package doctor

import (
	"github.com/lazynop/lazynf/internal/github"
)

// checkGitHub reports which auth source the github.Client picked. Does NOT
// hit the network; this is purely a configuration introspection.
func checkGitHub(c *github.Client) []Check {
	check := Check{Section: "GitHub auth", Title: "auth source"}
	src := c.AuthSource()
	if src == github.AuthNone {
		check.Severity = SeverityWarn
		check.Detail = "anonymous (subject to lower rate limits)"
		check.Hint = "set GITHUB_TOKEN or run `gh auth login` to authenticate"
		return []Check{check}
	}
	check.Severity = SeverityOK
	check.Detail = "using " + src.String()
	return []Check{check}
}
