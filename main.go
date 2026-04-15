// ABOUTME: CLI entry point for sprint-ledger TUI.
// ABOUTME: Parses args, loads sprint data, and starts the Bubbletea program.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sprint-ledger <project-root>")
		os.Exit(1)
	}
	fmt.Println("sprint-ledger:", os.Args[1])
}
