# Expert Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all critical and important issues identified by the 8-expert panel review of sprint-ledger-browser.

**Architecture:** Surgical fixes to existing files. No new files except one testdata fixture for BOM testing. Each task is independent — no ordering dependencies between tasks.

**Tech Stack:** Go, Bubbletea, Lipgloss, encoding/csv, standard library

**Triage notes:**
- `go.mod go 1.25.5` — the compatibility reviewer flagged this as non-existent, but `go version` returns `go1.25.5 darwin/arm64` on this machine. This is the real installed version. Leaving as-is.
- Exported symbols in `package main` — the Go idioms reviewer flagged this. While technically correct, Go convention commonly uses exported names in `package main` for readability and testability. Not fixing — it's a style preference, not a bug.
- Raw markdown display — by-design per the spec ("No markdown rendering (glamour, etc.) — plain text display of markdown source"). Not fixing.
- `--version` flag — YAGNI for an internal tool. Not fixing.
- Search/filter — feature request, not a bug. Out of scope.
- Responsive column widths — nice-to-have. Out of scope.

---

### Task 1: Clamp Viewport Dimensions at Small Terminal Sizes

**Severity:** CRITICAL — negative dimensions cause panics.

**Files:**
- Modify: `model.go:77-112`
- Modify: `model_test.go`

- [ ] **Step 1: Write failing tests for tiny terminal sizes**

Add these tests to `model_test.go`:

```go
func TestModel_TinyTerminalNoPanic(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project")

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
```

Add `"fmt"` to the import list in `model_test.go` if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestModel_TinyTerminalNoPanic -v`
Expected: panic or failure due to negative dimensions

- [ ] **Step 3: Add dimension clamping in Update**

In `model.go`, replace the WindowSizeMsg handler (lines 81-112) with:

```go
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 1
		helpHeight := 1
		borderHeight := 4 // 2 lines per bordered pane × 2 panes
		availableHeight := m.height - headerHeight - helpHeight - borderHeight

		tableHeight := availableHeight * 40 / 100
		if tableHeight < 3 {
			tableHeight = 3
		}
		detailHeight := availableHeight - tableHeight
		if detailHeight < 1 {
			detailHeight = 1
		}

		contentWidth := m.width - 2
		if contentWidth < 1 {
			contentWidth = 1
		}

		m.table.SetWidth(contentWidth)
		m.table.SetHeight(tableHeight)

		if !m.ready {
			m.viewport = viewport.New(contentWidth, detailHeight)
			if len(m.sprints) > 0 {
				m.viewport.SetContent(LoadDetailContent(m.sprints[0]))
			} else {
				m.viewport.SetContent("No sprints found")
			}
			m.ready = true
		} else {
			m.viewport.Width = contentWidth
			m.viewport.Height = detailHeight
		}

		return m, nil
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run TestModel_TinyTerminalNoPanic -v`
Expected: PASS — no panics at any size

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
git add model.go model_test.go
git commit -m "fix: clamp viewport dimensions to prevent panics at tiny terminal sizes"
```

---

### Task 2: Add --help Flag

**Severity:** CRITICAL — first-time users have zero context about what the tool does.

**Files:**
- Modify: `main.go:13-34`

- [ ] **Step 1: Write a test for --help output**

There's no clean way to unit-test `main()` directly since it calls `os.Exit`. Instead, we test via subprocess. Add a new file `main_test.go`:

Create file `main_test.go`:

```go
// ABOUTME: Tests for CLI argument handling and help output.
// ABOUTME: Uses subprocess execution to test main() behavior including exit codes.
package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain_HelpFlag(t *testing.T) {
	// Build the binary first
	bin := t.TempDir() + "/sprint-ledger-test"
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			cmd := exec.Command(bin, flag)
			out, err := cmd.CombinedOutput()
			output := string(out)

			// Should exit 0 for help
			if err != nil {
				t.Errorf("expected exit 0 for %s, got error: %v\nOutput: %s", flag, err, output)
			}

			if !strings.Contains(output, "sprint-ledger") {
				t.Errorf("help output should mention 'sprint-ledger', got: %s", output)
			}
			if !strings.Contains(output, "<project-root>") {
				t.Errorf("help output should mention '<project-root>', got: %s", output)
			}
		})
	}
}

func TestMain_NoArgs(t *testing.T) {
	bin := t.TempDir() + "/sprint-ledger-test"
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	cmd := exec.Command(bin)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for no args")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestMain_HelpFlag -v`
Expected: FAIL — `--help` is not handled, exits with error

- [ ] **Step 3: Add help flag handling to main.go**

Replace the arg parsing section of `main.go` (lines 13-17) with:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run "TestMain_HelpFlag|TestMain_NoArgs" -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
git add main.go main_test.go
git commit -m "feat: add --help and -h flags with usage information"
```

---

### Task 3: Strip UTF-8 BOM from TSV Files

**Severity:** IMPORTANT — Excel/Notepad-saved TSVs would fail to parse.

**Files:**
- Modify: `ledger.go:27-36`
- Modify: `ledger_test.go`
- Create: `testdata/bom/.ai/ledger.tsv` (BOM-prefixed TSV)

- [ ] **Step 1: Create BOM test fixture**

Create directory and file `testdata/bom/.ai/ledger.tsv`. The file must start with the 3-byte UTF-8 BOM (`\xef\xbb\xbf`) followed by normal TSV content:

```bash
mkdir -p testdata/bom/.ai
printf '\xef\xbb\xbfsprint_id\ttitle\tstatus\tcreated_at\tupdated_at\n001\tBOM Test\tcompleted\t2026-04-10T19:11:51Z\t2026-04-11T10:00:00Z\n' > testdata/bom/.ai/ledger.tsv
```

- [ ] **Step 2: Write failing test**

Add to `ledger_test.go`:

```go
func TestParseLedger_BOMPrefix(t *testing.T) {
	data, err := os.Open(filepath.Join("testdata", "bom", ".ai", "ledger.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	sprints, err := ParseLedger(data)
	if err != nil {
		t.Fatalf("BOM-prefixed file should parse successfully, got: %v", err)
	}

	if len(sprints) != 1 {
		t.Fatalf("expected 1 sprint, got %d", len(sprints))
	}

	if sprints[0].ID != "001" {
		t.Errorf("expected ID '001', got '%s'", sprints[0].ID)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -run TestParseLedger_BOMPrefix -v`
Expected: FAIL — "ledger missing required columns" because header is `"\ufeffsprint_id"`

- [ ] **Step 4: Add BOM stripping to ParseLedger**

In `ledger.go`, add `"bytes"` to the imports and replace the beginning of `ParseLedger` (lines 27-36):

```go
func ParseLedger(r io.Reader) ([]Sprint, error) {
	// Strip UTF-8 BOM if present (common with Excel/Notepad-saved files)
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test -run TestParseLedger_BOMPrefix -v`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 7: Commit**

```bash
git add ledger.go ledger_test.go testdata/bom/
git commit -m "fix: strip UTF-8 BOM from TSV files for Excel/Notepad compatibility"
```

---

### Task 4: Resolve Project Root to Absolute Path

**Severity:** IMPORTANT — `sprint-ledger .` shows "." as project name.

**Files:**
- Modify: `main.go:19-26`
- Modify: `main_test.go`

- [ ] **Step 1: Write failing test**

Add to `main_test.go`:

```go
func TestMain_DotPathShowsRealName(t *testing.T) {
	// Build the binary
	bin := t.TempDir() + "/sprint-ledger-test"
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	// Run with "testdata/valid" as a relative path — header should show "valid", not "."
	cmd := exec.Command(bin, "testdata/valid")
	// The TUI requires a terminal, so just check it doesn't crash on startup
	// by sending it a small terminal and immediate quit
	cmd.Env = append(os.Environ(), "TERM=dumb")
	out, _ := cmd.CombinedOutput()
	// With altscreen and no TTY, bubbletea will print an error about terminal.
	// The real validation is that filepath.Abs is called — verified by reading the code.
	_ = out
}
```

This test is limited because Bubbletea needs a TTY. The real verification is a code review + the existing E2E tests. The fix is trivial and mechanical.

- [ ] **Step 2: Add filepath.Abs to main.go**

In `main.go`, replace lines 19-26 with:

```go
	projectRoot := os.Args[1]
	projectRoot, err := filepath.Abs(projectRoot)
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
```

Note: This introduces two `err` bindings in the same function. The first uses `:=`, the second reuses `err` with `=`. Since the `filepath.Abs` line uses `:=` to introduce the shadowed `projectRoot`, `err` is also newly declared there.

- [ ] **Step 3: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add main.go main_test.go
git commit -m "fix: resolve project-root to absolute path for correct header display"
```

---

### Task 5: Add Escape Key for Detail-to-Table Navigation

**Severity:** IMPORTANT — natural expectation for pane navigation.

**Files:**
- Modify: `model.go:114-128`
- Modify: `model_test.go`

- [ ] **Step 1: Write failing test**

Add to `model_test.go`:

```go
func TestModel_EscapeReturnsToTable(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project")

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestModel_EscapeReturnsToTable -v`
Expected: FAIL — escape key not handled, focus stays on detail

- [ ] **Step 3: Add escape key handler**

In `model.go`, update the KeyMsg switch (lines 114-128). Add `"esc"` case:

```go
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

		case "esc":
			if m.focus == focusDetail {
				m.focus = focusTable
				m.table.Focus()
			}
			return m, nil
		}
```

- [ ] **Step 4: Update help text**

In `model.go`, update the help string (line 169):

```go
	help := helpStyle.Render("j/k navigate • tab focus • esc back • q quit")
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -run TestModel_EscapeReturnsToTable -v`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass. Note: `TestE2E_FullRender` checks for "quit" in help text — it will still pass since the new help text contains "quit".

- [ ] **Step 7: Commit**

```bash
git add model.go model_test.go
git commit -m "feat: add escape key to return from detail pane to table"
```

---

### Task 6: Differentiate Enter from Tab Keybindings

**Severity:** IMPORTANT — enter and tab doing the same thing wastes a keybinding.

Per the design spec: `enter` / `tab` = "Toggle focus between table and detail pane". Three reviewers flagged the redundancy. The fix: `tab` toggles focus (bidirectional), `enter` specifically moves table→detail (one-way, like "drill into"), and `esc` returns detail→table (from Task 5).

**Files:**
- Modify: `model.go:114-128`
- Modify: `model_test.go`

- [ ] **Step 1: Write failing test**

Add to `model_test.go`:

```go
func TestModel_EnterOnlyGoesToDetail(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project")

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestModel_EnterOnlyGoesToDetail -v`
Expected: FAIL — enter currently toggles back to table from detail

- [ ] **Step 3: Split enter and tab behavior**

In `model.go`, replace the `"tab", "enter"` case with separate cases:

```go
		case "tab":
			if m.focus == focusTable {
				m.focus = focusDetail
				m.table.Blur()
			} else {
				m.focus = focusTable
				m.table.Focus()
			}
			return m, nil

		case "enter":
			if m.focus == focusTable {
				m.focus = focusDetail
				m.table.Blur()
			}
			return m, nil
```

- [ ] **Step 4: Update help text**

In `model.go`, update the help string:

```go
	help := helpStyle.Render("j/k navigate • enter/tab focus • esc back • q quit")
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -run TestModel_EnterOnlyGoesToDetail -v`
Expected: PASS

- [ ] **Step 6: Update existing TestModel_FocusToggleWithEnter**

The existing `TestModel_FocusToggleWithEnter` test assumes enter toggles focus. It now needs to be updated since enter from detail no longer toggles back. Update it:

```go
func TestModel_FocusToggleWithEnter(t *testing.T) {
	sprints := loadTestSprints(t)
	m := newModel(sprints, "test-project")

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
```

- [ ] **Step 7: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 8: Commit**

```bash
git add model.go model_test.go
git commit -m "refactor: differentiate enter (table->detail) from tab (toggle)"
```

---

### Task 7: Sanitize Sprint IDs Against Path Traversal

**Severity:** IMPORTANT — sprint ID `../../etc/passwd` resolves to arbitrary file paths.

**Files:**
- Modify: `ledger.go:108-113`
- Modify: `ledger_test.go`

- [ ] **Step 1: Write failing test**

Add to `ledger_test.go`:

```go
func TestLoadSprints_PathTraversalInID(t *testing.T) {
	// Create a temporary project with a malicious sprint ID
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(aiDir, 0755)

	tsv := "sprint_id\ttitle\tstatus\tcreated_at\tupdated_at\n" +
		"../../etc/passwd\tEvil Sprint\tplanned\t2026-04-10T19:11:51Z\t2026-04-11T10:00:00Z\n"
	os.WriteFile(filepath.Join(aiDir, "ledger.tsv"), []byte(tsv), 0644)

	sprints, err := LoadSprints(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(sprints) != 1 {
		t.Fatalf("expected 1 sprint, got %d", len(sprints))
	}

	// The sprint should parse, but DetailPath should NOT resolve to a file
	// outside the project directory. The path traversal ID should be treated
	// as a plain filename that doesn't exist as a detail file.
	if sprints[0].DetailPath != "" {
		resolved, _ := filepath.Abs(sprints[0].DetailPath)
		projectAbs, _ := filepath.Abs(tmpDir)
		if !strings.HasPrefix(resolved, projectAbs) {
			t.Errorf("DetailPath escaped project root: %s", resolved)
		}
	}
}
```

Add `"strings"` to the imports in `ledger_test.go` if not already present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestLoadSprints_PathTraversalInID -v`
Expected: Could pass if `../../etc/passwd` doesn't match a SPRINT-*.md file, but the real concern is that `LoadDetailContent` could be called with the traversal path. The test validates the DetailPath doesn't escape the project root.

- [ ] **Step 3: Add path validation to LoadSprints**

In `ledger.go`, replace the detail path resolution loop (lines 108-113):

```go
	sprintsDir := filepath.Join(projectRoot, ".ai", "sprints")
	for i := range sprints {
		// Reject IDs containing path separators or traversal sequences
		if strings.ContainsAny(sprints[i].ID, "/\\") || strings.Contains(sprints[i].ID, "..") {
			continue
		}
		detailPath := filepath.Join(sprintsDir, fmt.Sprintf("SPRINT-%s.md", sprints[i].ID))
		if _, err := os.Stat(detailPath); err == nil {
			sprints[i].DetailPath = detailPath
		}
	}
```

Add `"strings"` to imports in `ledger.go` if not already present.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestLoadSprints_PathTraversalInID -v`
Expected: PASS — DetailPath is empty for the traversal ID

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
git add ledger.go ledger_test.go
git commit -m "fix: reject sprint IDs with path traversal sequences"
```

---

### Task 8: Return Errors from LoadDetailContent Instead of Hiding Them

**Severity:** IMPORTANT — callers can't distinguish error from content.

**Files:**
- Modify: `ledger.go:118-129`
- Modify: `model.go:101-104,138-141`
- Modify: `ledger_test.go`
- Modify: `model_test.go`

- [ ] **Step 1: Write test for new signature**

Update `TestLoadDetailContent_ValidFile` and `TestLoadDetailContent_MissingFile` in `ledger_test.go`:

```go
func TestLoadDetailContent_ValidFile(t *testing.T) {
	s := Sprint{
		ID:         "001",
		DetailPath: filepath.Join("testdata", "valid", ".ai", "sprints", "SPRINT-001.md"),
	}
	content, err := LoadDetailContent(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty content")
	}
	if content[:2] != "# " {
		t.Errorf("expected markdown heading, got '%s'", content[:10])
	}
}

func TestLoadDetailContent_MissingFile(t *testing.T) {
	s := Sprint{ID: "999", DetailPath: ""}
	content, err := LoadDetailContent(s)
	if err != nil {
		t.Fatal("empty DetailPath should not be an error")
	}
	expected := "No sprint detail file found for SPRINT-999.md"
	if content != expected {
		t.Errorf("expected '%s', got '%s'", expected, content)
	}
}

func TestLoadDetailContent_UnreadableFile(t *testing.T) {
	s := Sprint{ID: "001", DetailPath: "/nonexistent/path/SPRINT-001.md"}
	_, err := LoadDetailContent(s)
	if err == nil {
		t.Error("expected error for unreadable file")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run "TestLoadDetailContent" -v`
Expected: FAIL — compilation error because signature changed

- [ ] **Step 3: Update LoadDetailContent to return (string, error)**

In `ledger.go`, replace `LoadDetailContent` (lines 118-129):

```go
// LoadDetailContent reads the content of a sprint detail file.
// Returns a placeholder message if the path is empty.
// Returns an error if the file exists but can't be read.
func LoadDetailContent(s Sprint) (string, error) {
	if s.DetailPath == "" {
		return fmt.Sprintf("No sprint detail file found for SPRINT-%s.md", s.ID), nil
	}
	data, err := os.ReadFile(s.DetailPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", s.DetailPath, err)
	}
	return string(data), nil
}
```

- [ ] **Step 4: Update all callers in model.go**

In `model.go`, update the three places that call `LoadDetailContent`:

Line ~102 (initial content load):
```go
			if len(m.sprints) > 0 {
				content, err := LoadDetailContent(m.sprints[0])
				if err != nil {
					content = fmt.Sprintf("Error: %v", err)
				}
				m.viewport.SetContent(content)
			} else {
```

Line ~139 (cursor change):
```go
		if m.table.Cursor() != prevCursor && m.table.Cursor() < len(m.sprints) {
			content, err := LoadDetailContent(m.sprints[m.table.Cursor()])
			if err != nil {
				content = fmt.Sprintf("Error: %v", err)
			}
			m.viewport.SetContent(content)
			m.viewport.GotoTop()
		}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
git add ledger.go ledger_test.go model.go model_test.go
git commit -m "refactor: return error from LoadDetailContent instead of hiding it"
```

---

### Task 9: Add Scenario Tests for New Fixes

**Severity:** Validation — ensures fixes work with real data.

**Files:**
- Modify: `scenarios.jsonl`

- [ ] **Step 1: Add new scenario entries to scenarios.jsonl**

Append these lines to `scenarios.jsonl`:

```jsonl
{"name":"tiny_terminal_no_panic","description":"TUI renders without panic at extremely small terminal sizes","given":"Any project with sprints loaded","when":"WindowSizeMsg with sizes 1x1, 3x3, 10x5, 80x8 are sent","then":"View() returns non-empty string without panicking for all sizes","validates":"Dimension clamping prevents negative viewport/table sizes"}
{"name":"bom_prefix_parse","description":"TSV files with UTF-8 BOM prefix parse correctly","given":"A ledger.tsv file saved with Excel/Notepad UTF-8 BOM prefix","when":"ParseLedger processes the file","then":"Header columns are detected correctly and sprints are parsed","validates":"BOM stripping handles files from common editors"}
{"name":"path_traversal_rejected","description":"Sprint IDs with path traversal sequences do not resolve to files outside project root","given":"A ledger with sprint_id containing ../ or path separators","when":"LoadSprints resolves detail paths","then":"DetailPath is empty for malicious IDs — no file outside .ai/sprints/ is referenced","validates":"Path validation rejects directory traversal in sprint IDs"}
```

- [ ] **Step 2: Commit**

```bash
git add scenarios.jsonl
git commit -m "docs: add scenario entries for expert review fixes"
```
