// ABOUTME: File watcher for live-reloading the sprint ledger.
// ABOUTME: Uses fsnotify to detect ledger file changes and sends Bubbletea messages.
package main

import (
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// ledgerChangedMsg signals that the ledger file was modified on disk.
type ledgerChangedMsg struct{}

// watchLedger returns a tea.Cmd that watches the ledger file for changes.
// On a write event it sends a ledgerChangedMsg. Debounces rapid writes
// (editors often fire multiple events per save).
func watchLedger(projectRoot string) tea.Cmd {
	return func() tea.Msg {
		aiDir := filepath.Join(projectRoot, ".ai")

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}

		// Watch the .ai directory so we catch renames/creates too
		// (some editors do atomic saves via tmp file + rename)
		if err := watcher.Add(aiDir); err != nil {
			watcher.Close()
			return nil
		}

		defer watcher.Close()

		debounce := time.NewTimer(0)
		debounce.Stop()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				base := filepath.Base(event.Name)
				if base != "ledger.yaml" && base != "ledger.tsv" {
					continue
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
					debounce.Reset(100 * time.Millisecond)
				}
			case <-debounce.C:
				return ledgerChangedMsg{}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}
