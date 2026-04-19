// ABOUTME: Integration tests for the root Bubbletea model.
// ABOUTME: Tests model initialization, cursor movement, focus switching, and quit behavior.
package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func loadTestSprints(t *testing.T) []Sprint {
	t.Helper()
	sprints, err := LoadSprints(filepath.Join("testdata", "valid"))
	if err != nil {
		t.Fatal(err)
	}
	return sprints
}

func TestModel_Init(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	if m.projectName != "test-project" {
		t.Errorf("expected project name 'test-project', got '%s'", m.projectName)
	}
	if len(m.sprints) != 3 {
		t.Errorf("expected 3 sprints, got %d", len(m.sprints))
	}
	if m.focus != focusTable {
		t.Error("expected initial focus on table")
	}
}

func TestModel_WindowResize(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	um := updated.(model)

	if !um.ready {
		t.Error("expected model to be ready after resize")
	}
	if um.width != 80 {
		t.Errorf("expected width 80, got %d", um.width)
	}
}

func TestModel_QuitOnQ(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	// Send resize first so model is ready
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	updatedModel, cmd := updated.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_ = updatedModel

	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestModel_FocusToggle(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	// Resize to initialize viewport
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	if m2.focus != focusTable {
		t.Fatal("expected initial focus on table")
	}

	// Tab to switch to detail
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	m3 := updated.(model)

	if m3.focus != focusDetail {
		t.Error("expected focus on detail after tab")
	}

	// Tab again to switch back to table
	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyTab})
	m4 := updated.(model)

	if m4.focus != focusTable {
		t.Error("expected focus back on table after second tab")
	}
}

func TestE2E_FullRender(t *testing.T) {
	sprints, err := LoadSprints(filepath.Join("testdata", "valid"))
	if err != nil {
		t.Fatal(err)
	}

	m := newModel(sprints, "test-project", "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	view := m2.View()

	if !containsString(view, "test-project") {
		t.Error("view should contain project name")
	}

	if !containsString(view, "001") {
		t.Error("view should contain sprint ID 001")
	}
	if !containsString(view, "Project Scaffolding") {
		t.Error("view should contain sprint title")
	}

	if !containsString(view, "Sprint 001") {
		t.Error("view should contain detail content for sprint 001")
	}

	if !containsString(view, "quit") {
		t.Error("view should contain help text")
	}
}

func TestE2E_EmptyLedger(t *testing.T) {
	sprints, err := LoadSprints(filepath.Join("testdata", "empty"))
	if err != nil {
		t.Fatal(err)
	}

	m := newModel(sprints, "empty-project", "")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	view := m2.View()
	if !containsString(view, "No sprints found") {
		t.Error("empty ledger should show 'No sprints found'")
	}
}

func TestModel_CursorMovementUpdatesDetail(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	// Resize to initialize
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	// Initial view should show sprint 001 detail
	view1 := m2.View()
	if !containsString(view1, "Sprint 001") {
		t.Error("initial view should show Sprint 001 detail")
	}

	// Move cursor down (send "j" key while table focused)
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m3 := updated.(model)

	// View should now show sprint 002 detail
	view2 := m3.View()
	if !containsString(view2, "Sprint 002") {
		t.Error("after cursor down, view should show Sprint 002 detail")
	}
}

func TestModel_FocusToggleWithEnter(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	// Enter to switch to detail
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := updated.(model)

	if m3.focus != focusDetail {
		t.Error("expected focus on detail after enter")
	}

	// Enter again should stay in detail (enter is one-way: table->detail)
	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := updated.(model)

	if m4.focus != focusDetail {
		t.Error("expected focus to stay on detail after second enter")
	}
}

func TestModel_EnterOnlyGoesToDetail(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	// Enter while in table -> should go to detail
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := updated.(model)
	if m3.focus != focusDetail {
		t.Error("enter from table should go to detail")
	}

	// Enter while in detail -> should stay in detail (not toggle back)
	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := updated.(model)
	if m4.focus != focusDetail {
		t.Error("enter from detail should stay in detail")
	}
}

func TestModel_QuitOnCtrlC(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	_, cmd := updated.(model).Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Fatal("expected quit command from ctrl+c, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestModel_EscapeReturnsToTable(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	// Resize to initialize
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	// Tab to detail
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	m3 := updated.(model)
	if m3.focus != focusDetail {
		t.Fatal("expected focus on detail after tab")
	}

	// Escape to go back to table
	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m4 := updated.(model)

	if m4.focus != focusTable {
		t.Error("expected focus on table after escape")
	}
}

func TestModel_TinyTerminalNoPanic(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project", "")

	// Terminal so small that detailHeight would go negative without clamping
	sizes := []tea.WindowSizeMsg{
		{Width: 10, Height: 5},
		{Width: 1, Height: 1},
		{Width: 80, Height: 8},
		{Width: 3, Height: 3},
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			updated, _ := m.Update(size)
			m2 := updated.(model)

			// Should not panic when rendering
			view := m2.View()
			if len(view) == 0 {
				t.Error("expected non-empty view")
			}
		})
	}
}

// containsString checks if needle is in haystack, stripping ANSI escape codes first.
func containsString(haystack, needle string) bool {
	return strings.Contains(stripAnsi(haystack), needle)
}

func stripAnsi(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}
