// ABOUTME: File watcher for live-reloading sprint data across ledger and iter-* layouts.
// ABOUTME: Uses fsnotify; emits ledgerChangedMsg on any relevant change.
package main

import (
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// ledgerChangedMsg signals that watched files were modified on disk.
// (Name kept for back-compat with model.go; covers both layouts.)
type ledgerChangedMsg struct{}

type layout int

const (
	layoutLedger layout = iota
	layoutIter
)

// watchProject returns a tea.Cmd that watches the relevant files for the given
// layout. On a write/create/rename event it sends a ledgerChangedMsg.
// Debounces rapid writes (editors often fire multiple events per save).
func watchProject(projectRoot string, lay layout) tea.Cmd {
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}
		defer watcher.Close()

		dirs := watchDirs(projectRoot, lay)
		added := 0
		for _, d := range dirs {
			// Best-effort: missing directories are silently skipped.
			if err := watcher.Add(d); err == nil {
				added++
			}
		}
		if added == 0 {
			return nil
		}

		debounce := time.NewTimer(0)
		debounce.Stop()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if !isRelevantEvent(event, projectRoot, lay) {
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

// watchDirs returns the directories to add to the fsnotify watcher for a layout.
func watchDirs(projectRoot string, lay layout) []string {
	switch lay {
	case layoutLedger:
		return []string{filepath.Join(projectRoot, ".ai")}
	case layoutIter:
		return []string{
			filepath.Join(projectRoot, ".ai"),
			filepath.Join(projectRoot, ".ai", "iter-run-temp"),
			filepath.Join(projectRoot, ".ai", "iter-scope-temp"),
			filepath.Join(projectRoot, "docs", "iterations"),
		}
	}
	return nil
}

// isRelevantEvent decides whether an fsnotify event should trigger a reload.
func isRelevantEvent(event fsnotify.Event, projectRoot string, lay layout) bool {
	base := filepath.Base(event.Name)
	switch lay {
	case layoutLedger:
		return base == "ledger.yaml" || base == "ledger.tsv"
	case layoutIter:
		// Roadmap files and current-iter state files.
		switch base {
		case "roadmap.md", "iterations.md", "progress.md",
			"iter-current-iteration.txt", "iter-workflow-status.txt",
			"iter-loop-context.txt", "task_status.txt":
			return true
		}
		// Files inside iter-run-temp/: only .md and .txt.
		if strings.HasPrefix(event.Name, filepath.Join(projectRoot, ".ai", "iter-run-temp")) {
			return strings.HasSuffix(base, ".md") || strings.HasSuffix(base, ".txt")
		}
		return false
	}
	return false
}
