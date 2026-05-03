package ui

import "github.com/charmbracelet/lipgloss"

// Centralized palette so all CLI output is styled consistently.
// Keep simple: a handful of named styles, no theming surface yet.
var (
	StyleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true) // green
	StyleFailure = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // red
	StyleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true) // yellow
	StyleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))            // cyan
	StyleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))            // grey
	StyleBold    = lipgloss.NewStyle().Bold(true)
)
