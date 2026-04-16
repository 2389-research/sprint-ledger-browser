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
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		if len(os.Args) >= 2 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
			fmt.Println("sprint-ledger — browse .ai/ledger.tsv sprint data in a TUI")
			fmt.Println()
			fmt.Println("Usage: sprint-ledger <project-root>")
			fmt.Println()
			fmt.Println("Reads <project-root>/.ai/ledger.tsv and displays sprints in a")
			fmt.Println("navigable table with detail pane showing SPRINT-*.md content.")
			fmt.Println()
			fmt.Println("Keys: j/k navigate, tab toggle focus, q quit")
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "usage: sprint-ledger <project-root>")
		os.Exit(1)
	}

	projectRoot, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

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
