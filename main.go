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
	if len(os.Args) >= 2 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println("sprint-ledger — browse .ai/ sprint data in a TUI")
		fmt.Println()
		fmt.Println("Usage: sprint-ledger [project-root]")
		fmt.Println()
		fmt.Println("Reads <project-root>/.ai/ledger.yaml (or .tsv) and displays sprints")
		fmt.Println("in a navigable table with detail pane showing SPRINT-*.md content.")
		fmt.Println()
		fmt.Println("If no project-root is given, uses the current directory when")
		fmt.Println("it contains an .ai folder.")
		fmt.Println()
		fmt.Println("Keys: j/k navigate, enter/tab focus, esc back, q quit")
		os.Exit(0)
	}

	var arg string
	if len(os.Args) >= 2 {
		arg = os.Args[1]
	} else {
		// No arg — check if current directory has .ai/
		if _, err := os.Stat(filepath.Join(".", ".ai")); err == nil {
			arg = "."
		} else {
			fmt.Fprintln(os.Stderr, "usage: sprint-ledger [project-root]")
			fmt.Fprintln(os.Stderr, "  (or run from a directory containing .ai/)")
			os.Exit(1)
		}
	}

	projectRoot, err := filepath.Abs(arg)
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
