// ABOUTME: Lipgloss style definitions for the sprint-ledger TUI.
// ABOUTME: Defines border styles, focus states, header, and help text styles.
package main

import "github.com/charmbracelet/lipgloss"

var (
	// Border styles for focused/unfocused panes
	focusedBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	blurredBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8"))

	// Header style
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	// Help text at the bottom
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Padding(0, 1)
)
