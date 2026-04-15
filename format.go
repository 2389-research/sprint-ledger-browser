// ABOUTME: Display formatting for sprint data in the TUI table.
// ABOUTME: Converts status enums to colored icons and timestamps to short dates.
package main

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// FormatStatus returns a colored status icon and short label for display in the table.
func FormatStatus(status string) string {
	switch status {
	case "completed":
		return greenStyle.Render("● done")
	case "in_progress":
		return yellowStyle.Render("◐ wip")
	case "planned":
		return dimStyle.Render("○ plan")
	case "skipped":
		return dimStyle.Render("⊘ skip")
	default:
		return "? " + status
	}
}

// FormatDate returns a short date string like "Apr 13". Returns "—" for zero time.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("Jan 02")
}

// SprintsToRows converts a slice of Sprints to table row data.
// Each row is: [ID, Title, FormattedStatus, FormattedDate].
func SprintsToRows(sprints []Sprint) [][]string {
	rows := make([][]string, len(sprints))
	for i, s := range sprints {
		rows[i] = []string{
			s.ID,
			s.Title,
			FormatStatus(s.Status),
			FormatDate(s.UpdatedAt),
		}
	}
	return rows
}
