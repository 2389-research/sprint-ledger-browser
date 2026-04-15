# Sprint Ledger Browser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a read-only Bubbletea TUI that displays `.ai/ledger.tsv` sprint data in a split-pane view with navigable table and scrollable detail viewport.

**Architecture:** Single Go binary with a root Bubbletea model managing two child components: a `bubbles/table` for sprint rows and a `bubbles/viewport` for SPRINT-*.md content. All state is loaded from disk at startup; no live reload.

**Tech Stack:** Go, Bubbletea, Bubbles (table + viewport), Lipgloss

---

## File Structure

```
sprint-ledger-browser/
├── main.go            # CLI entry point: arg parsing, data loading, starts TUI program
├── ledger.go          # Sprint struct, TSV parser, detail file path resolution
├── ledger_test.go     # Unit tests for TSV parsing, status normalization, file resolution
├── format.go          # Status icon/color formatting, date formatting, sprint-to-row conversion
├── format_test.go     # Unit tests for formatting functions
├── styles.go          # Lipgloss style constants (borders, colors, focus states)
├── model.go           # Root Bubbletea model: Init, Update, View, layout, focus routing
├── model_test.go      # Integration tests for model Update behavior via teatest
├── testdata/
│   ├── valid/
│   │   └── .ai/
│   │       ├── ledger.tsv
│   │       └── sprints/
│   │           ├── SPRINT-001.md
│   │           ├── SPRINT-002.md
│   │           └── SPRINT-003.md
│   ├── empty/
│   │   └── .ai/
│   │       └── ledger.tsv          # header only
│   └── malformed/
│       └── .ai/
│           └── ledger.tsv          # mixed valid/invalid rows
├── go.mod
└── go.sum
```

**Responsibilities:**
- `ledger.go` — data layer. Parses TSV, resolves file paths. No TUI imports.
- `format.go` — presentation layer. Converts Sprint data to display strings. Imports lipgloss for coloring.
- `styles.go` — pure style definitions. No logic, just lipgloss constants.
- `model.go` — TUI orchestration. Owns the table and viewport, handles layout math and input routing.
- `main.go` — thin entry point. Validates args, calls ledger loader, passes data to model, runs program.

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`

- [ ] **Step 1: Initialize Go module**

Run:
```bash
go mod init sprint-ledger
```

- [ ] **Step 2: Add dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 3: Create minimal main.go**

```go
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
```

- [ ] **Step 4: Verify it compiles**

Run: `go build -o sprint-ledger .`
Expected: binary created, no errors

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "feat: scaffold Go module with dependencies"
```

---

### Task 2: Test Fixtures

**Files:**
- Create: `testdata/valid/.ai/ledger.tsv`
- Create: `testdata/valid/.ai/sprints/SPRINT-001.md`
- Create: `testdata/valid/.ai/sprints/SPRINT-002.md`
- Create: `testdata/valid/.ai/sprints/SPRINT-003.md`
- Create: `testdata/empty/.ai/ledger.tsv`
- Create: `testdata/malformed/.ai/ledger.tsv`

- [ ] **Step 1: Create valid test fixture**

`testdata/valid/.ai/ledger.tsv` (tab-separated):
```tsv
sprint_id	title	status	created_at	updated_at
001	Project Scaffolding	completed	2026-04-10T19:11:51Z	2026-04-11T10:00:00Z
002	Persistence Layer	in_progress	2026-04-10T19:11:51Z	2026-04-12T14:30:00Z
003	Slider Component	planned	2026-04-10T19:11:51Z	2026-04-10T19:11:51Z
```

`testdata/valid/.ai/sprints/SPRINT-001.md`:
```markdown
# Sprint 001 — Project Scaffolding

## Scope
- Initialize Go module
- Set up project structure
- Configure linting

## Definition of Done
- [ ] All files created
- [ ] Tests passing
```

`testdata/valid/.ai/sprints/SPRINT-002.md`:
```markdown
# Sprint 002 — Persistence Layer

## Scope
- Database schema design
- Repository pattern implementation
```

`testdata/valid/.ai/sprints/SPRINT-003.md`:
```markdown
# Sprint 003 — Slider Component

## Scope
- Build slider widget
- Wire up to form
```

- [ ] **Step 2: Create empty test fixture**

`testdata/empty/.ai/ledger.tsv`:
```tsv
sprint_id	title	status	created_at	updated_at
```

- [ ] **Step 3: Create malformed test fixture**

`testdata/malformed/.ai/ledger.tsv` (mix of valid, too-few-columns, and `complete` variant):
```tsv
sprint_id	title	status	created_at	updated_at
001	Good Row	complete	2026-04-10T19:11:51Z	2026-04-11T10:00:00Z
bad row with no tabs
003	Skipped Sprint	skipped	2026-04-10T19:11:51Z	2026-04-12T00:00:00Z
```

- [ ] **Step 4: Commit**

```bash
git add testdata/
git commit -m "test: add ledger TSV test fixtures"
```

---

### Task 3: Sprint Type + TSV Parser (TDD)

**Files:**
- Create: `ledger.go`
- Create: `ledger_test.go`

- [ ] **Step 1: Write failing test for ParseLedger happy path**

`ledger_test.go`:
```go
// ABOUTME: Unit tests for TSV ledger parsing and sprint detail file resolution.
// ABOUTME: Covers happy path, malformed input, status normalization, and missing files.
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseLedger_ValidFile(t *testing.T) {
	data, err := os.Open(filepath.Join("testdata", "valid", ".ai", "ledger.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	sprints, err := ParseLedger(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(sprints) != 3 {
		t.Fatalf("expected 3 sprints, got %d", len(sprints))
	}

	s := sprints[0]
	if s.ID != "001" {
		t.Errorf("expected ID '001', got '%s'", s.ID)
	}
	if s.Title != "Project Scaffolding" {
		t.Errorf("expected title 'Project Scaffolding', got '%s'", s.Title)
	}
	if s.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", s.Status)
	}
	expected := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	if !s.UpdatedAt.Equal(expected) {
		t.Errorf("expected UpdatedAt %v, got %v", expected, s.UpdatedAt)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestParseLedger_ValidFile -v`
Expected: FAIL — `ParseLedger` not defined

- [ ] **Step 3: Implement Sprint struct and ParseLedger**

`ledger.go`:
```go
// ABOUTME: Sprint data model and TSV ledger parser.
// ABOUTME: Reads .ai/ledger.tsv files and resolves paths to SPRINT-*.md detail files.
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Sprint struct {
	ID         string
	Title      string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DetailPath string
}

// ParseLedger reads a tab-separated ledger file and returns Sprint records.
// Skips the header row. Skips rows with wrong column count.
// Normalizes "complete" to "completed".
func ParseLedger(r io.Reader) ([]Sprint, error) {
	reader := csv.NewReader(r)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1 // allow variable fields so we can skip bad rows
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading TSV: %w", err)
	}

	var sprints []Sprint
	for i, row := range records {
		if i == 0 {
			continue // skip header
		}
		if len(row) != 5 {
			continue // skip malformed rows
		}

		status := row[2]
		if status == "complete" {
			status = "completed"
		}

		createdAt, err := time.Parse(time.RFC3339, row[3])
		if err != nil {
			createdAt = time.Time{}
		}
		updatedAt, err := time.Parse(time.RFC3339, row[4])
		if err != nil {
			updatedAt = time.Time{}
		}

		sprints = append(sprints, Sprint{
			ID:        row[0],
			Title:     row[1],
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	return sprints, nil
}

// LoadSprints reads the ledger from a project root and resolves detail file paths.
func LoadSprints(projectRoot string) ([]Sprint, error) {
	ledgerPath := filepath.Join(projectRoot, ".ai", "ledger.tsv")
	f, err := os.Open(ledgerPath)
	if err != nil {
		return nil, fmt.Errorf("opening ledger: %w", err)
	}
	defer f.Close()

	sprints, err := ParseLedger(f)
	if err != nil {
		return nil, err
	}

	for i := range sprints {
		detailPath := filepath.Join(projectRoot, ".ai", "sprints", fmt.Sprintf("SPRINT-%s.md", sprints[i].ID))
		if _, err := os.Stat(detailPath); err == nil {
			sprints[i].DetailPath = detailPath
		}
	}

	return sprints, nil
}

// LoadDetailContent reads the content of a sprint detail file.
// Returns a placeholder message if the path is empty or the file can't be read.
func LoadDetailContent(s Sprint) string {
	if s.DetailPath == "" {
		return fmt.Sprintf("No sprint detail file found for SPRINT-%s.md", s.ID)
	}
	data, err := os.ReadFile(s.DetailPath)
	if err != nil {
		return fmt.Sprintf("Error reading %s: %v", s.DetailPath, err)
	}
	return string(data)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestParseLedger_ValidFile -v`
Expected: PASS

- [ ] **Step 5: Write failing test for malformed input + status normalization**

Add to `ledger_test.go`:
```go
func TestParseLedger_MalformedRows(t *testing.T) {
	data, err := os.Open(filepath.Join("testdata", "malformed", ".ai", "ledger.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	sprints, err := ParseLedger(data)
	if err != nil {
		t.Fatal(err)
	}

	// Should skip the "bad row with no tabs" line, keep 2 valid rows
	if len(sprints) != 2 {
		t.Fatalf("expected 2 sprints (skipping malformed), got %d", len(sprints))
	}

	// "complete" should be normalized to "completed"
	if sprints[0].Status != "completed" {
		t.Errorf("expected 'completed' (normalized), got '%s'", sprints[0].Status)
	}

	if sprints[1].Status != "skipped" {
		t.Errorf("expected 'skipped', got '%s'", sprints[1].Status)
	}
}

func TestParseLedger_EmptyLedger(t *testing.T) {
	data, err := os.Open(filepath.Join("testdata", "empty", ".ai", "ledger.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	sprints, err := ParseLedger(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(sprints) != 0 {
		t.Fatalf("expected 0 sprints, got %d", len(sprints))
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test -run TestParseLedger -v`
Expected: all 3 tests PASS

- [ ] **Step 7: Write failing test for LoadSprints**

Add to `ledger_test.go`:
```go
func TestLoadSprints_ResolvesDetailPaths(t *testing.T) {
	sprints, err := LoadSprints(filepath.Join("testdata", "valid"))
	if err != nil {
		t.Fatal(err)
	}

	if len(sprints) != 3 {
		t.Fatalf("expected 3 sprints, got %d", len(sprints))
	}

	for _, s := range sprints {
		if s.DetailPath == "" {
			t.Errorf("sprint %s: expected DetailPath to be set", s.ID)
		}
	}
}

func TestLoadSprints_MissingLedger(t *testing.T) {
	_, err := LoadSprints(filepath.Join("testdata", "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing ledger, got nil")
	}
}

func TestLoadDetailContent_ValidFile(t *testing.T) {
	s := Sprint{
		ID:         "001",
		DetailPath: filepath.Join("testdata", "valid", ".ai", "sprints", "SPRINT-001.md"),
	}
	content := LoadDetailContent(s)
	if len(content) == 0 {
		t.Error("expected non-empty content")
	}
	if content[:2] != "# " {
		t.Errorf("expected markdown heading, got '%s'", content[:10])
	}
}

func TestLoadDetailContent_MissingFile(t *testing.T) {
	s := Sprint{ID: "999", DetailPath: ""}
	content := LoadDetailContent(s)
	expected := "No sprint detail file found for SPRINT-999.md"
	if content != expected {
		t.Errorf("expected '%s', got '%s'", expected, content)
	}
}
```

- [ ] **Step 8: Run all tests**

Run: `go test -v`
Expected: all tests PASS

- [ ] **Step 9: Commit**

```bash
git add ledger.go ledger_test.go
git commit -m "feat: add Sprint type, TSV parser, and detail file loader"
```

---

### Task 4: Formatting Helpers (TDD)

**Files:**
- Create: `format.go`
- Create: `format_test.go`

- [ ] **Step 1: Write failing test for FormatStatus**

`format_test.go`:
```go
// ABOUTME: Unit tests for sprint display formatting functions.
// ABOUTME: Covers status icons, date formatting, and table row conversion.
package main

import (
	"strings"
	"testing"
	"time"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status   string
		wantIcon string
	}{
		{"completed", "●"},
		{"in_progress", "◐"},
		{"planned", "○"},
		{"skipped", "⊘"},
		{"unknown", "?"},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			result := FormatStatus(tc.status)
			if !strings.Contains(result, tc.wantIcon) {
				t.Errorf("FormatStatus(%q) = %q, want icon %q", tc.status, result, tc.wantIcon)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	ts := time.Date(2026, 4, 13, 18, 10, 39, 0, time.UTC)
	result := FormatDate(ts)
	if result != "Apr 13" {
		t.Errorf("expected 'Apr 13', got '%s'", result)
	}
}

func TestFormatDate_Zero(t *testing.T) {
	result := FormatDate(time.Time{})
	if result != "—" {
		t.Errorf("expected '—' for zero time, got '%s'", result)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run "TestFormat" -v`
Expected: FAIL — `FormatStatus`, `FormatDate` not defined

- [ ] **Step 3: Implement formatting functions**

`format.go`:
```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run "TestFormat" -v`
Expected: all PASS

- [ ] **Step 5: Write failing test for SprintsToRows**

Add to `format_test.go`:
```go
func TestSprintsToRows(t *testing.T) {
	sprints := []Sprint{
		{ID: "001", Title: "Scaffolding", Status: "completed", UpdatedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)},
		{ID: "002", Title: "Persistence", Status: "planned", UpdatedAt: time.Time{}},
	}

	rows := SprintsToRows(sprints)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "001" {
		t.Errorf("expected row[0][0] = '001', got '%s'", rows[0][0])
	}
	if rows[1][3] != "—" {
		t.Errorf("expected '—' for zero date, got '%s'", rows[1][3])
	}
}
```

- [ ] **Step 6: Run all format tests**

Run: `go test -run "TestFormat\|TestSprintsToRows" -v`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add format.go format_test.go
git commit -m "feat: add status/date formatting and row conversion"
```

---

### Task 5: Lipgloss Styles

**Files:**
- Create: `styles.go`

- [ ] **Step 1: Define all style constants**

`styles.go`:
```go
// ABOUTME: Lipgloss style definitions for the sprint-ledger TUI.
// ABOUTME: Defines border styles, focus states, header, and help text styles.
package main

import "github.com/charmbracelet/lipgloss"

var (
	// Border styles for focused/unfocused panes
	focusedBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	blurredBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8"))

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
```

- [ ] **Step 2: Verify it compiles**

Run: `go build .`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add styles.go
git commit -m "feat: add lipgloss style definitions"
```

---

### Task 6: Root TUI Model

**Files:**
- Create: `model.go`

This is the main Bubbletea model that wires table + viewport together. Implements `tea.Model`.

- [ ] **Step 1: Create the root model with Init, Update, View**

`model.go`:
```go
// ABOUTME: Root Bubbletea model for the sprint-ledger TUI.
// ABOUTME: Manages split-pane layout with table (top) and detail viewport (bottom).
package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusTable focus = iota
	focusDetail
)

type model struct {
	sprints     []Sprint
	projectName string
	table       table.Model
	viewport    viewport.Model
	focus       focus
	width       int
	height      int
	ready       bool
}

func newModel(sprints []Sprint, projectName string) model {
	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Title", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Updated", Width: 8},
	}

	rawRows := SprintsToRows(sprints)
	rows := make([]table.Row, len(rawRows))
	for i, r := range rawRows {
		rows[i] = table.Row(r)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(5),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{
		sprints:     sprints,
		projectName: projectName,
		table:       t,
		focus:       focusTable,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 1
		helpHeight := 1
		availableHeight := m.height - headerHeight - helpHeight

		tableHeight := availableHeight * 40 / 100
		if tableHeight < 3 {
			tableHeight = 3
		}
		detailHeight := availableHeight - tableHeight

		m.table.SetWidth(m.width - 2)  // account for borders
		m.table.SetHeight(tableHeight)

		if !m.ready {
			m.viewport = viewport.New(m.width-2, detailHeight)
			if len(m.sprints) > 0 {
				m.viewport.SetContent(LoadDetailContent(m.sprints[0]))
			} else {
				m.viewport.SetContent("No sprints found")
			}
			m.ready = true
		} else {
			m.viewport.Width = m.width - 2
			m.viewport.Height = detailHeight
		}

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab", "enter":
			if m.focus == focusTable {
				m.focus = focusDetail
				m.table.Blur()
			} else {
				m.focus = focusTable
				m.table.Focus()
			}
			return m, nil
		}
	}

	prevCursor := m.table.Cursor()

	if m.focus == focusTable {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)

		// If cursor moved, update detail content
		if m.table.Cursor() != prevCursor && m.table.Cursor() < len(m.sprints) {
			m.viewport.SetContent(LoadDetailContent(m.sprints[m.table.Cursor()]))
			m.viewport.GotoTop()
		}
	} else {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "loading..."
	}

	header := headerStyle.Width(m.width).Render(
		fmt.Sprintf("sprint-ledger: %s", m.projectName),
	)

	var tableView, detailView string
	if m.focus == focusTable {
		tableView = focusedBorder.Width(m.width - 2).Render(m.table.View())
		detailView = blurredBorder.Width(m.width - 2).Render(m.viewport.View())
	} else {
		tableView = blurredBorder.Width(m.width - 2).Render(m.table.View())
		detailView = focusedBorder.Width(m.width - 2).Render(m.viewport.View())
	}

	help := helpStyle.Render("↑/↓ navigate • tab focus • q quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tableView,
		detailView,
		help,
	)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build .`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add model.go
git commit -m "feat: add root Bubbletea model with split-pane layout"
```

---

### Task 7: CLI Entry Point

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Wire main.go to load data and start TUI**

Replace `main.go` with:
```go
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
```

- [ ] **Step 2: Build and smoke test with test fixtures**

Run:
```bash
go build -o sprint-ledger .
./sprint-ledger testdata/valid
```
Expected: TUI launches with 3 sprints, navigable with j/k, detail pane shows markdown, tab switches focus, q quits.

- [ ] **Step 3: Test error case**

Run: `./sprint-ledger testdata/nonexistent`
Expected: prints `error: opening ledger: ...` to stderr, exits 1

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire CLI entry point to load data and start TUI"
```

---

### Task 8: Integration + E2E Tests

**Files:**
- Create: `model_test.go`

- [ ] **Step 1: Write integration test for model initialization**

`model_test.go`:
```go
// ABOUTME: Integration tests for the root Bubbletea model.
// ABOUTME: Tests model initialization, cursor movement, focus switching, and quit behavior.
package main

import (
	"path/filepath"
	"testing"
	"time"

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
	m := newModel(sprints, "test-project")

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
	m := newModel(sprints, "test-project")

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
	m := newModel(sprints, "test-project")

	// Send resize first so model is ready
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	updated, cmd := m.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_ = updated

	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	// Execute the command and check it produces a QuitMsg
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestModel_FocusToggle(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project")

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
```

- [ ] **Step 2: Run integration tests**

Run: `go test -run "TestModel" -v`
Expected: all PASS

- [ ] **Step 3: Write E2E test verifying full load-to-render pipeline**

Add to `model_test.go`:
```go
func TestE2E_FullRender(t *testing.T) {
	sprints, err := LoadSprints(filepath.Join("testdata", "valid"))
	if err != nil {
		t.Fatal(err)
	}

	m := newModel(sprints, "test-project")

	// Simulate window resize to trigger layout
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	view := m2.View()

	// Header should contain project name
	if !containsString(view, "test-project") {
		t.Error("view should contain project name")
	}

	// Table should contain sprint data
	if !containsString(view, "001") {
		t.Error("view should contain sprint ID 001")
	}
	if !containsString(view, "Project Scaffolding") {
		t.Error("view should contain sprint title")
	}

	// Detail pane should show first sprint's content
	if !containsString(view, "Sprint 001") {
		t.Error("view should contain detail content for sprint 001")
	}

	// Help text should be present
	if !containsString(view, "quit") {
		t.Error("view should contain help text")
	}
}

func TestE2E_EmptyLedger(t *testing.T) {
	sprints, err := LoadSprints(filepath.Join("testdata", "empty"))
	if err != nil {
		t.Fatal(err)
	}

	m := newModel(sprints, "empty-project")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(model)

	view := m2.View()
	if !containsString(view, "No sprints found") {
		t.Error("empty ledger should show 'No sprints found'")
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		// Simple substring check; ANSI codes may be present so we
		// strip them for comparison
		stripAnsi(haystack) != "" &&
		indexOf(stripAnsi(haystack), needle) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func stripAnsi(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find the terminal letter
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			if j < len(s) {
				j++ // skip the terminal letter
			}
			i = j
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}

// Unused import guard — time is used in other test files in same package
var _ = time.Now
```

- [ ] **Step 4: Run all tests**

Run: `go test -v`
Expected: all tests PASS (unit, integration, and E2E)

- [ ] **Step 5: Commit**

```bash
git add model_test.go
git commit -m "test: add integration and E2E tests for TUI model"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `go test -v -count=1 ./...`
Expected: all tests PASS

- [ ] **Step 2: Build clean binary**

Run: `go build -o sprint-ledger .`
Expected: binary created with no warnings

- [ ] **Step 3: Test against real project data**

Run: `./sprint-ledger /Users/harper/workspace/2389/caralanay/energy-tracker`
Expected: TUI shows 11 sprints from the energy-tracker project. Navigate with j/k, tab to detail pane, scroll markdown, q to quit.

- [ ] **Step 4: Test against empty/missing cases**

Run:
```bash
./sprint-ledger testdata/empty     # should show "No sprints found"
./sprint-ledger /nonexistent       # should print error and exit 1
```

- [ ] **Step 5: Run go vet**

Run: `go vet ./...`
Expected: no issues

- [ ] **Step 6: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "chore: final cleanup after verification"
```
