package doctor

import (
	"fmt"
	"os/exec"
	"runtime"
)

// checkFcCache verifies fontconfig's fc-cache binary is reachable on Linux.
// On macOS/Windows the check is a noop (CoreText / Windows registry handle
// font discovery without an external invocation).
func checkFcCache() []Check {
	c := Check{Section: SectionFcCache, Title: "fc-cache"}
	if runtime.GOOS != "linux" {
		c.Severity = SeverityOK
		c.Detail = fmt.Sprintf("not applicable on %s", runtime.GOOS)
		return []Check{c}
	}
	path, err := exec.LookPath("fc-cache")
	if err != nil {
		c.Severity = SeverityWarn
		c.Detail = "fc-cache not found in PATH"
		c.Hint = "install fontconfig (apt install fontconfig / dnf install fontconfig)"
		return []Check{c}
	}
	c.Severity = SeverityOK
	c.Detail = fmt.Sprintf("found at %s", path)
	return []Check{c}
}
