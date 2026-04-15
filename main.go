// ABOUTME: CLI entry point for sprint-ledger TUI.
// ABOUTME: Parses args, loads sprint data, and starts the Bubbletea program.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sprint-ledger <project-root>")
		os.Exit(1)
	}

	projectRoot := os.Args[1]
	sprints, err := LoadSprints(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectName := filepath.Base(projectRoot)
	m := newModel(sprints, projectName)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running TUI: %v\n", err)
		os.Exit(1)
	}
}
