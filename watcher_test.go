// ABOUTME: Tests for ledger file watching and live reload.
// ABOUTME: Verifies that modifying the ledger file triggers a reload message.
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWatchLedger_DetectsChange(t *testing.T) {
	// Set up a temp project with a ledger file
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(aiDir, 0755)

	ledgerPath := filepath.Join(aiDir, "ledger.tsv")
	initial := "sprint_id\ttitle\tstatus\n001\tFirst\tplanned\n"
	os.WriteFile(ledgerPath, []byte(initial), 0644)

	// Start watching in a goroutine
	cmd := watchProject(tmpDir, layoutLedger)

	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	// Give the watcher time to start
	time.Sleep(200 * time.Millisecond)

	// Modify the ledger file
	updated := "sprint_id\ttitle\tstatus\n001\tFirst\tcompleted\n002\tSecond\tplanned\n"
	os.WriteFile(ledgerPath, []byte(updated), 0644)

	// Wait for the message (with timeout)
	select {
	case msg := <-done:
		if _, ok := msg.(ledgerChangedMsg); !ok {
			t.Errorf("expected ledgerChangedMsg, got %T", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ledgerChangedMsg")
	}
}

func TestWatchLedger_IgnoresOtherFiles(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(aiDir, 0755)

	ledgerPath := filepath.Join(aiDir, "ledger.tsv")
	os.WriteFile(ledgerPath, []byte("sprint_id\ttitle\tstatus\n"), 0644)

	cmd := watchProject(tmpDir, layoutLedger)

	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	time.Sleep(200 * time.Millisecond)

	// Write a non-ledger file — should not trigger
	os.WriteFile(filepath.Join(aiDir, "notes.md"), []byte("hello"), 0644)

	select {
	case msg := <-done:
		t.Errorf("expected no message for non-ledger file, got %T", msg)
	case <-time.After(500 * time.Millisecond):
		// Good — no message received
	}
}

func TestModel_LedgerChangedMsg_ReloadsData(t *testing.T) {
	// Set up a temp project
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	sprintsDir := filepath.Join(aiDir, "sprints")
	os.MkdirAll(sprintsDir, 0755)

	tsv := "sprint_id\ttitle\tstatus\n001\tFirst Sprint\tplanned\n"
	os.WriteFile(filepath.Join(aiDir, "ledger.tsv"), []byte(tsv), 0644)
	os.WriteFile(filepath.Join(sprintsDir, "SPRINT-001.md"), []byte("# First"), 0644)

	sprints, _ := LoadSprints(tmpDir)
	m := newModel(sprints, "test", tmpDir, layoutLedger)

	// Initialize and resize
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	if len(m2.sprints) != 1 {
		t.Fatalf("expected 1 sprint, got %d", len(m2.sprints))
	}

	// Add a second sprint to the ledger on disk
	tsv2 := "sprint_id\ttitle\tstatus\n001\tFirst Sprint\tcompleted\n002\tSecond Sprint\tplanned\n"
	os.WriteFile(filepath.Join(aiDir, "ledger.tsv"), []byte(tsv2), 0644)

	// Send ledgerChangedMsg
	updated, cmd := m2.Update(ledgerChangedMsg{})
	m3 := updated.(model)

	if len(m3.sprints) != 2 {
		t.Fatalf("after reload expected 2 sprints, got %d", len(m3.sprints))
	}
	if m3.sprints[0].Status != "completed" {
		t.Errorf("expected sprint 001 status 'completed', got '%s'", m3.sprints[0].Status)
	}

	// Should return a new watch command to re-arm the watcher
	if cmd == nil {
		t.Error("expected a new watchLedger cmd after reload")
	}
}

func TestWatchProject_LedgerLayout_BehavesAsBefore(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(aiDir, 0755)
	ledgerPath := filepath.Join(aiDir, "ledger.tsv")
	os.WriteFile(ledgerPath, []byte("sprint_id\ttitle\tstatus\n001\tFirst\tplanned\n"), 0644)

	cmd := watchProject(tmpDir, layoutLedger)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	time.Sleep(200 * time.Millisecond)

	os.WriteFile(ledgerPath, []byte("sprint_id\ttitle\tstatus\n001\tFirst\tcompleted\n"), 0644)

	select {
	case msg := <-done:
		if _, ok := msg.(ledgerChangedMsg); !ok {
			t.Errorf("expected ledgerChangedMsg, got %T", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ledgerChangedMsg")
	}
}
