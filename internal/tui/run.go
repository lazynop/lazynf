// Package tui exposes the entry point used by cmd/root.go to launch the
// interactive Bubble Tea program.
package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lazynop/lazynf/internal/engine"
	"github.com/lazynop/lazynf/internal/tui/app"
)

// Run boots the bubbletea program and blocks until the user quits. The alt
// screen is requested by the root model's View itself; see app.Model.View.
func Run(eng *engine.Engine) error {
	m := app.New(eng)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
