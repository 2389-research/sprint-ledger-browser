# iter-* Layout Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `sprint-ledger-browser` work against the iter-* layout used by `kanmail` and `door` (no ledger file; iterations parsed from `docs/iterations/roadmap.md` with rich live overlay for the current iteration).

**Architecture:** Inline branch — extend `Sprint` struct with three optional fields, add a new `roadmap.go` parser, add a third fallback branch in `LoadSprints` (yaml → tsv → roadmap.md → `.ai/iter-scope-temp/iterations.md`). `LoadDetailContent` gains a `projectRoot` parameter and a new branch that composes a status-first overlay (progress, tasks, reviews, loop context, roadmap section) for the current iteration. Watcher becomes layout-aware.

**Tech Stack:** Go 1.22+, Bubbletea, lipgloss, fsnotify, gopkg.in/yaml.v3. Existing patterns: top-level `package main`, table-driven tests, file fixtures under `testdata/`.

**Spec:** `docs/superpowers/specs/2026-05-19-iter-layout-support-design.md`

**Branch:** `feat/iter-layout-support` (already created).

---

## Task 1: Extend Sprint struct with iter-* fields

**Files:**
- Modify: `ledger.go:18-25`

- [ ] **Step 1: Update Sprint struct**

Replace the existing `Sprint` struct (lines 18-25 of `ledger.go`) with:

```go
type Sprint struct {
	ID         string
	IDPrefix   string    // "SPRINT" (default) or "ITER"; empty treated as "SPRINT"
	Title      string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DetailPath string    // populated for ledger layout
	Section    string    // populated for iter-* layout — verbatim roadmap section
	IsCurrent  bool      // matches .ai/iter-current-iteration.txt
}
```

- [ ] **Step 2: Add effectivePrefix helper**

Append to `ledger.go`:

```go
// effectivePrefix returns the ID prefix for display, defaulting to "SPRINT"
// when Sprint.IDPrefix is empty (back-compat for ledger-layout sprints).
func effectivePrefix(s Sprint) string {
	if s.IDPrefix == "" {
		return "SPRINT"
	}
	return s.IDPrefix
}
```

- [ ] **Step 3: Run existing tests to confirm no regressions**

Run: `go test ./...`
Expected: PASS — all existing tests green; new fields are zero-valued and unused.

- [ ] **Step 4: Commit**

```bash
git add ledger.go
git commit -m "feat: extend Sprint struct with iter-* layout fields"
```

---

## Task 2: Add pending + unknown status formatting

**Files:**
- Modify: `format.go:18-33`

- [ ] **Step 1: Write failing test**

Append to `format_test.go`:

```go
func TestFormatStatus_PendingAndUnknown(t *testing.T) {
	cases := map[string]string{
		"pending": "○ pend",
		"unknown": "? unknown",
	}
	for input, wantSubstr := range cases {
		got := FormatStatus(input)
		if !strings.Contains(got, wantSubstr) {
			t.Errorf("FormatStatus(%q) = %q, want substring %q", input, got, wantSubstr)
		}
	}
}
```

Add the `"strings"` import if not already present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestFormatStatus_PendingAndUnknown -v`
Expected: FAIL — `"pending"` currently falls through to the default `"? pending"` branch.

- [ ] **Step 3: Add "pending" case to FormatStatus**

In `format.go`, add a case before the `default`:

```go
	case "pending":
		return dimStyle.Render("○ pend")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestFormatStatus -v`
Expected: PASS — both new cases handled.

- [ ] **Step 5: Commit**

```bash
git add format.go format_test.go
git commit -m "feat: format pending status with same icon as planned"
```

---

## Task 3: ParseRoadmap — basic heading and section extraction

**Files:**
- Create: `roadmap.go`
- Create: `roadmap_test.go`
- Create: `testdata/roadmap/basic.md`

- [ ] **Step 1: Create fixture**

Create `testdata/roadmap/basic.md`:

```markdown
# Roadmap

### ITER-0001 — First iteration

**Status:** done

Some body content for the first iteration.

### ITER-0002 — Second iteration

**Status:** pending

Body for second.

### ITER-0003a — Third sub-iteration

**Status:** in_progress

Sub-iter body.
```

- [ ] **Step 2: Write failing test**

Create `roadmap_test.go`:

```go
// ABOUTME: Unit tests for the roadmap.md iteration parser.
// ABOUTME: Covers heading detection, status normalization, and section preservation.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRoadmap_BasicHeadings(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "roadmap", "basic.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sprints, err := ParseRoadmap(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(sprints) != 3 {
		t.Fatalf("expected 3 sprints, got %d", len(sprints))
	}

	wantIDs := []string{"0001", "0002", "0003a"}
	wantTitles := []string{"First iteration", "Second iteration", "Third sub-iteration"}
	for i, s := range sprints {
		if s.ID != wantIDs[i] {
			t.Errorf("[%d] ID = %q, want %q", i, s.ID, wantIDs[i])
		}
		if s.Title != wantTitles[i] {
			t.Errorf("[%d] Title = %q, want %q", i, s.Title, wantTitles[i])
		}
		if s.IDPrefix != "ITER" {
			t.Errorf("[%d] IDPrefix = %q, want %q", i, s.IDPrefix, "ITER")
		}
		if !strings.Contains(s.Section, "### ITER-"+s.ID) {
			t.Errorf("[%d] Section missing heading; got: %q", i, s.Section)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -run TestParseRoadmap_BasicHeadings -v`
Expected: FAIL — `ParseRoadmap` undefined.

- [ ] **Step 4: Create roadmap.go with ParseRoadmap**

Create `roadmap.go`:

```go
// ABOUTME: Parser for iteration roadmap markdown files.
// ABOUTME: Reads docs/iterations/roadmap.md style files and returns iteration sprints.
package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// iterHeadingRE matches "### ITER-<id> <dash> <title>".
// Accepts em-dash (—), en-dash (–), and ASCII hyphen (-).
var iterHeadingRE = regexp.MustCompile(`^### ITER-(\S+)\s+[\x{2014}\x{2013}\-]\s+(.*)$`)

// statusLineRE captures the value after "**Status:**".
var statusLineRE = regexp.MustCompile(`^\*\*Status:\*\*\s*(.*)$`)

// ParseRoadmap reads a roadmap markdown file and returns one Sprint per
// "### ITER-..." heading, each carrying its raw section text in Section.
// Status normalization and current-iteration inference happen in LoadSprints,
// not here.
func ParseRoadmap(r io.Reader) ([]Sprint, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var sprints []Sprint
	var current *Sprint
	var section strings.Builder

	flush := func() {
		if current == nil {
			return
		}
		current.Section = strings.TrimRight(section.String(), "\n")
		sprints = append(sprints, *current)
		current = nil
		section.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()

		if m := iterHeadingRE.FindStringSubmatch(line); m != nil {
			flush()
			current = &Sprint{
				ID:       m[1],
				IDPrefix: "ITER",
				Title:    strings.TrimSpace(m[2]),
			}
			section.WriteString(line)
			section.WriteString("\n")
			continue
		}

		// A different ### heading ends the current section.
		if strings.HasPrefix(line, "### ") && current != nil {
			flush()
			continue
		}

		if current != nil {
			section.WriteString(line)
			section.WriteString("\n")
		}
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning roadmap: %w", err)
	}
	return sprints, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test -run TestParseRoadmap_BasicHeadings -v`
Expected: PASS — 3 sprints with correct IDs, titles, IDPrefix=ITER, sections contain heading.

- [ ] **Step 6: Commit**

```bash
git add roadmap.go roadmap_test.go testdata/roadmap/basic.md
git commit -m "feat: parse roadmap markdown into Sprint records"
```

---

## Task 4: ParseRoadmap — status normalization

**Files:**
- Modify: `roadmap.go`
- Modify: `roadmap_test.go`

- [ ] **Step 1: Write failing test**

Append to `roadmap_test.go`:

```go
func TestParseRoadmap_StatusNormalization(t *testing.T) {
	input := `### ITER-0001 — A
**Status:** done

### ITER-0002 — B
**Status:** pending

### ITER-0003 — C
**Status:** in_progress

### ITER-0004 — D
**Status:** done:ITER-0004

### ITER-0005 — E
**Status:** in_progress (ITER-0005a done:2026-05-15 — work; ITER-0005b pending — more)

### ITER-0006 — F
**Status:** fail

### ITER-0007 — G
**Status:** weird-value
`

	sprints, err := ParseRoadmap(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"0001": "completed",
		"0002": "pending",
		"0003": "in_progress",
		"0004": "completed",
		"0005": "in_progress",
		"0006": "failed",
		"0007": "unknown",
	}
	if len(sprints) != len(want) {
		t.Fatalf("expected %d sprints, got %d", len(want), len(sprints))
	}
	for _, s := range sprints {
		if got := s.Status; got != want[s.ID] {
			t.Errorf("ID %s: status = %q, want %q", s.ID, got, want[s.ID])
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestParseRoadmap_StatusNormalization -v`
Expected: FAIL — Status fields are all empty (no normalization yet).

- [ ] **Step 3: Add status extraction inside the scanner loop**

In `roadmap.go`, after the `// A different ### heading...` block and before the final `if current != nil` block, add:

```go
		if current != nil && current.Status == "" {
			if m := statusLineRE.FindStringSubmatch(line); m != nil {
				current.Status = normalizeStatus(m[1])
			}
		}
```

Also add the `normalizeStatus` function at the bottom of `roadmap.go`:

```go
// normalizeStatus extracts the leading token from a raw status string and maps
// it to a canonical status. Examples:
//
//	"done"                            -> "completed"
//	"done:ITER-0001"                  -> "completed"
//	"in_progress (ITER-0002a done…)"  -> "in_progress"
//	"fail"                            -> "failed"
//	"pending"                         -> "pending"
//	anything else                     -> "unknown"
func normalizeStatus(raw string) string {
	raw = strings.TrimSpace(raw)
	// Leading token: stop at first space, "(", or ":".
	idx := strings.IndexAny(raw, " (:")
	token := raw
	if idx >= 0 {
		token = raw[:idx]
	}
	switch token {
	case "done":
		return "completed"
	case "pending":
		return "pending"
	case "in_progress":
		return "in_progress"
	case "fail":
		return "failed"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestParseRoadmap_StatusNormalization -v`
Expected: PASS — all seven status variants map correctly.

- [ ] **Step 5: Commit**

```bash
git add roadmap.go roadmap_test.go
git commit -m "feat: normalize roadmap status tokens"
```

---

## Task 5: ParseRoadmap — dash variants and edge cases

**Files:**
- Modify: `roadmap_test.go`
- Create: `testdata/roadmap/dashes.md`
- Create: `testdata/roadmap/no-status.md`

- [ ] **Step 1: Create dash-variant fixture**

Create `testdata/roadmap/dashes.md`:

```markdown
### ITER-0001 — Em-dash title

**Status:** done

### ITER-0002 – En-dash title

**Status:** done

### ITER-0003 - Hyphen title

**Status:** done
```

- [ ] **Step 2: Create no-status fixture**

Create `testdata/roadmap/no-status.md`:

```markdown
### ITER-0001 — Alpha

Body without status.

### ITER-0002 — Bravo

Body without status.

### ITER-0003 — Charlie

Body without status.
```

- [ ] **Step 3: Write tests for dash variants and empty Status**

Append to `roadmap_test.go`:

```go
func TestParseRoadmap_DashVariants(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "roadmap", "dashes.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sprints, err := ParseRoadmap(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 3 {
		t.Fatalf("expected 3 sprints (em/en/hyphen), got %d", len(sprints))
	}
	wantTitles := []string{"Em-dash title", "En-dash title", "Hyphen title"}
	for i, s := range sprints {
		if s.Title != wantTitles[i] {
			t.Errorf("[%d] Title = %q, want %q", i, s.Title, wantTitles[i])
		}
	}
}

func TestParseRoadmap_NoStatusLines(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "roadmap", "no-status.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sprints, err := ParseRoadmap(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 3 {
		t.Fatalf("expected 3 sprints, got %d", len(sprints))
	}
	for _, s := range sprints {
		if s.Status != "" {
			t.Errorf("ID %s: expected empty Status (inference happens in LoadSprints), got %q", s.ID, s.Status)
		}
	}
}

func TestParseRoadmap_Empty(t *testing.T) {
	sprints, err := ParseRoadmap(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 0 {
		t.Errorf("expected 0 sprints, got %d", len(sprints))
	}
}

func TestParseRoadmap_SubIterIDs(t *testing.T) {
	input := `### ITER-0006a-i — Hyphenated sub-iter
**Status:** pending

### ITER-0009b — Lettered sub-iter
**Status:** pending
`
	sprints, err := ParseRoadmap(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 2 {
		t.Fatalf("expected 2 sprints, got %d", len(sprints))
	}
	if sprints[0].ID != "0006a-i" {
		t.Errorf("expected ID 0006a-i, got %q", sprints[0].ID)
	}
	if sprints[1].ID != "0009b" {
		t.Errorf("expected ID 0009b, got %q", sprints[1].ID)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run TestParseRoadmap -v`
Expected: PASS — all parser tests green (including the existing ones).

- [ ] **Step 5: Commit**

```bash
git add roadmap_test.go testdata/roadmap/dashes.md testdata/roadmap/no-status.md
git commit -m "test: cover dash variants, missing status, and sub-iter IDs"
```

---

## Task 6: LoadDetailContent signature change (mechanical refactor)

**Files:**
- Modify: `ledger.go:199-208` (LoadDetailContent)
- Modify: `model.go:109`, `model.go:149`, `model.go:207`
- Modify: `ledger_test.go:116`, `ledger_test.go:130`, `ledger_test.go:142`

This is a mechanical refactor — no behavior change. Existing tests must keep passing.

- [ ] **Step 1: Change LoadDetailContent signature in ledger.go**

Replace `ledger.go` lines 199-208 (the `LoadDetailContent` function) with:

```go
// LoadDetailContent reads the content of a sprint detail file.
// Returns a placeholder message if the path is empty.
// Returns an error if the file exists but can't be read.
// projectRoot is used for iter-* layout overlay reads; ignored for ledger layout.
func LoadDetailContent(s Sprint, projectRoot string) (string, error) {
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

- [ ] **Step 2: Update model.go call sites**

In `model.go` line 109 (in `case ledgerChangedMsg`), change:

```go
				content, err := LoadDetailContent(sprints[cursor])
```

to:

```go
				content, err := LoadDetailContent(sprints[cursor], m.projectRoot)
```

In `model.go` line 149 (in `tea.WindowSizeMsg` initial-load block), change:

```go
				content, err := LoadDetailContent(m.sprints[0])
```

to:

```go
				content, err := LoadDetailContent(m.sprints[0], m.projectRoot)
```

In `model.go` line 207 (in cursor-move block), change:

```go
			content, err := LoadDetailContent(m.sprints[m.table.Cursor()])
```

to:

```go
			content, err := LoadDetailContent(m.sprints[m.table.Cursor()], m.projectRoot)
```

- [ ] **Step 3: Update ledger_test.go call sites**

In `ledger_test.go`, find every `LoadDetailContent(s)` call (lines 116, 130, 142) and change to `LoadDetailContent(s, "")` — the existing tests don't need a real projectRoot because they exercise ledger-layout paths only.

- [ ] **Step 4: Run all tests to verify no regression**

Run: `go test ./...`
Expected: PASS — all existing tests green; signature change is mechanical.

- [ ] **Step 5: Commit**

```bash
git add ledger.go model.go ledger_test.go
git commit -m "refactor: thread projectRoot through LoadDetailContent"
```

---

## Task 7: LoadIterRoadmap helper with current-iter detection and inference

**Files:**
- Create: `iter.go`
- Modify: `roadmap_test.go`

- [ ] **Step 1: Write failing test**

Append to `roadmap_test.go`:

```go
func TestLoadIterRoadmap_InfersStatusFromCurrentIter(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	if err := os.MkdirAll(aiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	roadmap := `### ITER-0001 — A

Body.

### ITER-0002 — B

Body.

### ITER-0003 — C

Body.

### ITER-0004 — D

Body.
`
	if err := os.WriteFile(filepath.Join(docsDir, "roadmap.md"), []byte(roadmap), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aiDir, "iter-current-iteration.txt"), []byte("ITER-0002\n"), 0644); err != nil {
		t.Fatal(err)
	}

	sprints, err := LoadIterRoadmap(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 4 {
		t.Fatalf("expected 4 sprints, got %d", len(sprints))
	}

	wantStatus := []string{"completed", "in_progress", "pending", "pending"}
	wantCurrent := []bool{false, true, false, false}
	for i, s := range sprints {
		if s.Status != wantStatus[i] {
			t.Errorf("[%d] Status = %q, want %q", i, s.Status, wantStatus[i])
		}
		if s.IsCurrent != wantCurrent[i] {
			t.Errorf("[%d] IsCurrent = %v, want %v", i, s.IsCurrent, wantCurrent[i])
		}
	}
}

func TestLoadIterRoadmap_ExplicitStatusWinsOverInference(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	os.MkdirAll(aiDir, 0755)
	os.MkdirAll(docsDir, 0755)

	roadmap := `### ITER-0001 — A
**Status:** done

### ITER-0002 — B
**Status:** pending

### ITER-0003 — C
`
	os.WriteFile(filepath.Join(docsDir, "roadmap.md"), []byte(roadmap), 0644)
	// Current iter is ITER-0003 (last); ITER-0002 has explicit "pending"
	// which should NOT be overridden by inference even though it's before current.
	os.WriteFile(filepath.Join(aiDir, "iter-current-iteration.txt"), []byte("ITER-0003"), 0644)

	sprints, err := LoadIterRoadmap(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if sprints[1].Status != "pending" {
		t.Errorf("explicit pending should win, got %q", sprints[1].Status)
	}
}

func TestLoadIterRoadmap_FallbackPath(t *testing.T) {
	tmpDir := t.TempDir()
	scopeDir := filepath.Join(tmpDir, ".ai", "iter-scope-temp")
	os.MkdirAll(scopeDir, 0755)

	roadmap := `### ITER-0001 — Fallback
**Status:** done
`
	os.WriteFile(filepath.Join(scopeDir, "iterations.md"), []byte(roadmap), 0644)

	sprints, err := LoadIterRoadmap(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 1 || sprints[0].ID != "0001" {
		t.Fatalf("expected 1 sprint from fallback path, got %v", sprints)
	}
}

func TestLoadIterRoadmap_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadIterRoadmap(tmpDir)
	if err == nil {
		t.Error("expected error when neither roadmap path exists")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestLoadIterRoadmap -v`
Expected: FAIL — `LoadIterRoadmap` undefined.

- [ ] **Step 3: Create iter.go**

Create `iter.go`:

```go
// ABOUTME: Loader and detail composer for the iter-* project layout.
// ABOUTME: Reads docs/iterations/roadmap.md plus .ai/iter-* state files.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// roadmapSearchOrder is the fixed list of relative paths LoadIterRoadmap tries.
var roadmapSearchOrder = []string{
	filepath.Join("docs", "iterations", "roadmap.md"),
	filepath.Join(".ai", "iter-scope-temp", "iterations.md"),
}

// LoadIterRoadmap finds and parses a roadmap, then performs the current-iter
// detection and status-inference pass. Returns an error if no roadmap is found.
func LoadIterRoadmap(projectRoot string) ([]Sprint, error) {
	var (
		sprints []Sprint
		err     error
		found   bool
	)
	for _, rel := range roadmapSearchOrder {
		path := filepath.Join(projectRoot, rel)
		f, openErr := os.Open(path)
		if openErr != nil {
			continue
		}
		sprints, err = ParseRoadmap(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		found = true
		break
	}
	if !found {
		return nil, fmt.Errorf("no roadmap found (looked at %v relative to %s)", roadmapSearchOrder, projectRoot)
	}

	currentID := readCurrentIterationID(projectRoot)
	applyCurrentAndInference(sprints, currentID)
	return sprints, nil
}

// readCurrentIterationID reads .ai/iter-current-iteration.txt and strips the
// "ITER-" prefix. Returns "" if the file is missing or empty.
func readCurrentIterationID(projectRoot string) string {
	data, err := os.ReadFile(filepath.Join(projectRoot, ".ai", "iter-current-iteration.txt"))
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(string(data))
	return strings.TrimPrefix(raw, "ITER-")
}

// applyCurrentAndInference mutates sprints in place:
//   - sets IsCurrent on the sprint whose ID matches currentID
//   - fills empty Status fields based on position relative to the current iter
//
// If currentID is empty (no marker file) and Status is empty, leaves it "unknown".
func applyCurrentAndInference(sprints []Sprint, currentID string) {
	currentIdx := -1
	for i := range sprints {
		if sprints[i].ID == currentID {
			currentIdx = i
			sprints[i].IsCurrent = true
		}
	}
	for i := range sprints {
		if sprints[i].Status != "" {
			continue
		}
		switch {
		case currentIdx < 0:
			sprints[i].Status = "unknown"
		case i < currentIdx:
			sprints[i].Status = "completed"
		case i == currentIdx:
			sprints[i].Status = "in_progress"
		default:
			sprints[i].Status = "pending"
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run TestLoadIterRoadmap -v`
Expected: PASS — all four LoadIterRoadmap tests green.

- [ ] **Step 5: Commit**

```bash
git add iter.go roadmap_test.go
git commit -m "feat: load roadmap with current-iter detection and inference"
```

---

## Task 8: Integrate iter-* branch into LoadSprints

**Files:**
- Modify: `ledger.go:157-194` (LoadSprints)
- Modify: `ledger_test.go`

- [ ] **Step 1: Write failing integration test**

Append to `ledger_test.go`:

```go
func TestLoadSprints_IterLayout(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(docsDir, 0755)
	os.MkdirAll(aiDir, 0755)

	roadmap := `### ITER-0001 — First
**Status:** done

### ITER-0002 — Second

### ITER-0003 — Third
`
	os.WriteFile(filepath.Join(docsDir, "roadmap.md"), []byte(roadmap), 0644)
	os.WriteFile(filepath.Join(aiDir, "iter-current-iteration.txt"), []byte("ITER-0002"), 0644)

	sprints, err := LoadSprints(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 3 {
		t.Fatalf("expected 3 sprints, got %d", len(sprints))
	}
	if sprints[0].Status != "completed" || sprints[1].Status != "in_progress" || sprints[2].Status != "pending" {
		t.Errorf("status inference wrong: %q, %q, %q",
			sprints[0].Status, sprints[1].Status, sprints[2].Status)
	}
	if !sprints[1].IsCurrent {
		t.Error("expected ITER-0002 to be marked IsCurrent")
	}
	if sprints[0].IDPrefix != "ITER" {
		t.Errorf("expected IDPrefix ITER, got %q", sprints[0].IDPrefix)
	}
}

func TestLoadSprints_PrefersLedgerOverRoadmap(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(docsDir, 0755)
	os.MkdirAll(aiDir, 0755)

	os.WriteFile(filepath.Join(docsDir, "roadmap.md"),
		[]byte("### ITER-0001 — Should not be picked\n**Status:** done\n"), 0644)

	tsv := "sprint_id\ttitle\tstatus\nXYZ\tShould be picked\tplanned\n"
	os.WriteFile(filepath.Join(aiDir, "ledger.tsv"), []byte(tsv), 0644)

	sprints, err := LoadSprints(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 1 || sprints[0].ID != "XYZ" {
		t.Fatalf("expected ledger to win, got %v", sprints)
	}
	if sprints[0].IDPrefix == "ITER" {
		t.Errorf("expected ledger sprint to have empty/SPRINT IDPrefix, got %q", sprints[0].IDPrefix)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestLoadSprints_(IterLayout|PrefersLedgerOverRoadmap)' -v`
Expected: FAIL — current `LoadSprints` errors when no `ledger.{yaml,tsv}` is present.

- [ ] **Step 3: Add iter-* fallback branch to LoadSprints**

In `ledger.go`, replace the existing `LoadSprints` function (lines 157-194) with:

```go
// LoadSprints reads sprint records from a project root, trying layouts in order:
//
//  1. .ai/ledger.yaml          (existing yaml ledger)
//  2. .ai/ledger.tsv           (existing tsv ledger)
//  3. docs/iterations/roadmap.md or .ai/iter-scope-temp/iterations.md
//     (new iter-* layout)
//
// For ledger layouts, resolves per-sprint detail file paths under .ai/sprints/.
// For the iter-* layout, populates Section and IsCurrent in each Sprint.
func LoadSprints(projectRoot string) ([]Sprint, error) {
	aiDir := filepath.Join(projectRoot, ".ai")

	yamlPath := filepath.Join(aiDir, "ledger.yaml")
	if f, err := os.Open(yamlPath); err == nil {
		defer f.Close()
		sprints, err := ParseYAMLLedger(f)
		if err != nil {
			return nil, err
		}
		resolveDetailPaths(sprints, projectRoot)
		return sprints, nil
	}

	tsvPath := filepath.Join(aiDir, "ledger.tsv")
	if f, err := os.Open(tsvPath); err == nil {
		defer f.Close()
		sprints, err := ParseLedger(f)
		if err != nil {
			return nil, err
		}
		resolveDetailPaths(sprints, projectRoot)
		return sprints, nil
	}

	// Try iter-* layout last.
	if sprints, err := LoadIterRoadmap(projectRoot); err == nil {
		return sprints, nil
	}

	return nil, fmt.Errorf("opening ledger: no ledger.yaml, ledger.tsv, or roadmap.md found for %s", projectRoot)
}

// resolveDetailPaths populates DetailPath for ledger-layout sprints.
func resolveDetailPaths(sprints []Sprint, projectRoot string) {
	sprintsDir := filepath.Join(projectRoot, ".ai", "sprints")
	for i := range sprints {
		if strings.ContainsAny(sprints[i].ID, "/\\") || strings.Contains(sprints[i].ID, "..") {
			continue
		}
		detailPath := filepath.Join(sprintsDir, fmt.Sprintf("SPRINT-%s.md", sprints[i].ID))
		if _, err := os.Stat(detailPath); err == nil {
			sprints[i].DetailPath = detailPath
		}
	}
}
```

- [ ] **Step 4: Run all tests to verify no regression**

Run: `go test ./...`
Expected: PASS — new tests green; all existing ledger tests still green.

- [ ] **Step 5: Commit**

```bash
git add ledger.go ledger_test.go
git commit -m "feat: fall back to iter-* layout when no ledger file present"
```

---

## Task 9: LoadDetailContent — non-current iter renders Section verbatim

**Files:**
- Modify: `ledger.go` (LoadDetailContent)
- Modify: `ledger_test.go`

- [ ] **Step 1: Write failing test**

Append to `ledger_test.go`:

```go
func TestLoadDetailContent_IterNonCurrent(t *testing.T) {
	s := Sprint{
		ID:       "0001",
		IDPrefix: "ITER",
		Title:    "First",
		Status:   "completed",
		Section:  "### ITER-0001 — First\n**Status:** done\n\nBody.",
	}
	got, err := LoadDetailContent(s, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != s.Section {
		t.Errorf("expected Section verbatim, got: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestLoadDetailContent_IterNonCurrent -v`
Expected: FAIL — current `LoadDetailContent` returns the empty-DetailPath placeholder instead of the Section.

- [ ] **Step 3: Add iter-* branch to LoadDetailContent**

In `ledger.go`, replace `LoadDetailContent` with:

```go
// LoadDetailContent returns the detail markdown for a sprint.
//   - Ledger layout: reads s.DetailPath, or returns a placeholder when empty.
//   - Iter-* layout (s.Section non-empty): returns the roadmap section.
//     For the current iteration, prepends a live overlay.
func LoadDetailContent(s Sprint, projectRoot string) (string, error) {
	if s.Section != "" {
		if s.IsCurrent {
			return renderIterDetailWithOverlay(s, projectRoot), nil
		}
		return s.Section, nil
	}
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

- [ ] **Step 4: Add stub renderIterDetailWithOverlay in iter.go**

Append to `iter.go`:

```go
// renderIterDetailWithOverlay composes the status-first overlay for the
// current iteration: header + progress + tasks + reviews + loop context +
// roadmap section. Each block is best-effort; missing files are silently
// omitted. Implemented incrementally; see Task 10–12 for sections.
func renderIterDetailWithOverlay(s Sprint, projectRoot string) string {
	// Stub: just returns the section for now. Overlay blocks added in Task 10+.
	return s.Section
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -run TestLoadDetailContent -v`
Expected: PASS — non-current iter test passes, existing ledger tests still pass.

- [ ] **Step 6: Commit**

```bash
git add ledger.go iter.go ledger_test.go
git commit -m "feat: render iter-* section verbatim for non-current iterations"
```

---

## Task 10: Overlay composer — header, progress, and tasks blocks

**Files:**
- Modify: `iter.go` (renderIterDetailWithOverlay)
- Modify: `ledger_test.go`

- [ ] **Step 1: Write failing test**

Append to `ledger_test.go`:

```go
func TestLoadDetailContent_IterCurrent_HeaderAndBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	runTemp := filepath.Join(aiDir, "iter-run-temp")
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	os.MkdirAll(runTemp, 0755)
	os.MkdirAll(docsDir, 0755)

	os.WriteFile(filepath.Join(aiDir, "iter-workflow-status.txt"), []byte("fail\n"), 0644)
	os.WriteFile(filepath.Join(docsDir, "progress.md"),
		[]byte("# Iteration Progress\n\n**Iteration:** ITER-0003a\n**Tasks:** 8/12 done\n"), 0644)
	os.WriteFile(filepath.Join(runTemp, "task_status.txt"),
		[]byte("TASK-01|done\nTASK-02|in_progress\n"), 0644)

	s := Sprint{
		ID:        "0003a",
		IDPrefix:  "ITER",
		Title:     "Phase 2a",
		Status:    "in_progress",
		IsCurrent: true,
		Section:   "### ITER-0003a — Phase 2a\n**Status:** in_progress\n\nBody.",
	}
	got, err := LoadDetailContent(s, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Header line: ID + status + run status
	if !strings.HasPrefix(got, "ITER-0003a") {
		t.Errorf("expected header to start with ITER-0003a, got: %q", got[:40])
	}
	if !strings.Contains(got, "in_progress") {
		t.Errorf("expected parsed status in header")
	}
	if !strings.Contains(got, "run status: fail") {
		t.Errorf("expected run status in header")
	}

	// Progress block
	if !strings.Contains(got, "## Progress") {
		t.Errorf("expected Progress section header")
	}
	if !strings.Contains(got, "Tasks:** 8/12 done") {
		t.Errorf("expected progress.md content in output")
	}

	// Tasks block
	if !strings.Contains(got, "## Tasks") {
		t.Errorf("expected Tasks section header")
	}
	if !strings.Contains(got, "TASK-01|done") {
		t.Errorf("expected task_status content")
	}

	// Section still present at the bottom
	if !strings.Contains(got, "## Roadmap Section") {
		t.Errorf("expected Roadmap Section header")
	}
	if !strings.Contains(got, "### ITER-0003a — Phase 2a") {
		t.Errorf("expected roadmap section content")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestLoadDetailContent_IterCurrent_HeaderAndBlocks -v`
Expected: FAIL — overlay stub returns Section only.

- [ ] **Step 3: Implement header + progress + tasks blocks in iter.go**

Replace the `renderIterDetailWithOverlay` stub with:

```go
// renderIterDetailWithOverlay composes the status-first overlay for the
// current iteration: header + progress + tasks + reviews + loop context +
// roadmap section. Each block is best-effort; missing files are silently
// omitted.
func renderIterDetailWithOverlay(s Sprint, projectRoot string) string {
	var b strings.Builder

	// Header: "ITER-0003a  [in_progress | run status: fail]"
	b.WriteString(effectivePrefix(s))
	b.WriteString("-")
	b.WriteString(s.ID)
	b.WriteString("  [")
	b.WriteString(s.Status)
	if runStatus := readTrimmed(filepath.Join(projectRoot, ".ai", "iter-workflow-status.txt")); runStatus != "" {
		b.WriteString(" | run status: ")
		b.WriteString(runStatus)
	}
	b.WriteString("]\n\n")

	// Progress block (docs/iterations/progress.md) — only if it references the current ITER ID.
	progressPath := filepath.Join(projectRoot, "docs", "iterations", "progress.md")
	if progress, ok := readProgressIfMatches(progressPath, effectivePrefix(s)+"-"+s.ID); ok {
		b.WriteString("## Progress\n\n")
		b.WriteString(progress)
		ensureTrailingBlank(&b)
	}

	// Tasks block (.ai/iter-run-temp/task_status.txt)
	if tasks := readTrimmed(filepath.Join(projectRoot, ".ai", "iter-run-temp", "task_status.txt")); tasks != "" {
		b.WriteString("## Tasks\n\n")
		b.WriteString(tasks)
		b.WriteString("\n\n")
	}

	// Roadmap section last.
	b.WriteString("## Roadmap Section\n\n")
	b.WriteString(s.Section)
	return b.String()
}

// readTrimmed reads a file and returns its trimmed contents, or "" on any error.
func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// readProgressIfMatches returns the file contents only when the first ~20 lines
// reference the given fullID (e.g. "ITER-0003a"). Prevents stale snapshots
// showing the wrong iteration.
func readProgressIfMatches(path, fullID string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	content := string(data)
	head := content
	if nl := indexNthByte(content, '\n', 20); nl > 0 {
		head = content[:nl]
	}
	if !strings.Contains(head, fullID) {
		return "", false
	}
	return strings.TrimSpace(content), true
}

// indexNthByte returns the byte offset of the n-th occurrence of c, or -1 if
// it occurs fewer than n times.
func indexNthByte(s string, c byte, n int) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}

// ensureTrailingBlank guarantees the builder ends with "\n\n".
func ensureTrailingBlank(b *strings.Builder) {
	s := b.String()
	if !strings.HasSuffix(s, "\n\n") {
		if strings.HasSuffix(s, "\n") {
			b.WriteString("\n")
		} else {
			b.WriteString("\n\n")
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestLoadDetailContent_IterCurrent_HeaderAndBlocks -v`
Expected: PASS — header, progress, tasks, and roadmap-section all present.

- [ ] **Step 5: Commit**

```bash
git add iter.go ledger_test.go
git commit -m "feat: render header, progress, and tasks blocks in iter overlay"
```

---

## Task 11: Overlay composer — reviews and loop context blocks

**Files:**
- Modify: `iter.go` (renderIterDetailWithOverlay)
- Modify: `ledger_test.go`

- [ ] **Step 1: Write failing test**

Append to `ledger_test.go`:

```go
func TestLoadDetailContent_IterCurrent_ReviewsAndLoopContext(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	runTemp := filepath.Join(aiDir, "iter-run-temp")
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	os.MkdirAll(runTemp, 0755)
	os.MkdirAll(docsDir, 0755)

	os.WriteFile(filepath.Join(aiDir, "iter-loop-context.txt"),
		[]byte("Attempt 1 failed.\nReason: scope too large.\n"), 0644)
	os.WriteFile(filepath.Join(runTemp, "scope_review_a.md"), []byte("review a"), 0644)
	os.WriteFile(filepath.Join(runTemp, "scope_review_b.md"), []byte("review b"), 0644)
	os.WriteFile(filepath.Join(runTemp, "fresh_spec_a.md"), []byte("spec a"), 0644)
	os.WriteFile(filepath.Join(runTemp, "scope.md"), []byte("scope"), 0644)
	os.WriteFile(filepath.Join(runTemp, "ignored.log"), []byte("ignored"), 0644)

	s := Sprint{
		ID: "0003a", IDPrefix: "ITER", IsCurrent: true,
		Status: "in_progress", Section: "### ITER-0003a — Body",
	}
	got, err := LoadDetailContent(s, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "## Reviews") {
		t.Errorf("expected Reviews section header")
	}
	for _, name := range []string{"scope_review_a.md", "scope_review_b.md", "fresh_spec_a.md", "scope.md"} {
		if !strings.Contains(got, name) {
			t.Errorf("expected review filename %s in output", name)
		}
	}
	if strings.Contains(got, "ignored.log") {
		t.Errorf("unexpected non-review file listed: ignored.log")
	}
	if !strings.Contains(got, "## Loop Context") {
		t.Errorf("expected Loop Context section header")
	}
	if !strings.Contains(got, "Attempt 1 failed.") {
		t.Errorf("expected loop-context content")
	}

	// Reviews must appear before Loop Context, which must appear before Roadmap Section.
	idxReviews := strings.Index(got, "## Reviews")
	idxLoop := strings.Index(got, "## Loop Context")
	idxRoadmap := strings.Index(got, "## Roadmap Section")
	if !(idxReviews < idxLoop && idxLoop < idxRoadmap) {
		t.Errorf("ordering wrong: reviews=%d loop=%d roadmap=%d", idxReviews, idxLoop, idxRoadmap)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestLoadDetailContent_IterCurrent_ReviewsAndLoopContext -v`
Expected: FAIL — no Reviews or Loop Context blocks emitted yet.

- [ ] **Step 3: Add Reviews and Loop Context blocks**

In `iter.go`, in `renderIterDetailWithOverlay`, insert these blocks AFTER the Tasks block and BEFORE the Roadmap Section block:

```go
	// Reviews block — list filenames + mtimes only.
	if reviews := listReviewFiles(filepath.Join(projectRoot, ".ai", "iter-run-temp")); len(reviews) > 0 {
		b.WriteString("## Reviews\n\n")
		for _, r := range reviews {
			fmt.Fprintf(&b, "%s  (last modified %s)\n", r.name, r.mtime.Format("2006-01-02 15:04"))
		}
		b.WriteString("\n")
	}

	// Loop Context block.
	if ctx := readTrimmed(filepath.Join(projectRoot, ".ai", "iter-loop-context.txt")); ctx != "" {
		b.WriteString("## Loop Context\n\n")
		b.WriteString(ctx)
		b.WriteString("\n\n")
	}
```

Append helper types and functions to `iter.go`:

```go
type reviewFile struct {
	name  string
	mtime time.Time
}

// listReviewFiles returns review-related files in iter-run-temp, sorted by name.
// Matches: *_review_*.md, fresh_spec_*.md, scope.md. Excludes everything else.
func listReviewFiles(runTempDir string) []reviewFile {
	entries, err := os.ReadDir(runTempDir)
	if err != nil {
		return nil
	}
	var out []reviewFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isReviewFilename(name) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, reviewFile{name: name, mtime: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func isReviewFilename(name string) bool {
	if name == "scope.md" {
		return true
	}
	if strings.HasSuffix(name, ".md") {
		if strings.Contains(name, "_review_") {
			return true
		}
		if strings.HasPrefix(name, "fresh_spec_") {
			return true
		}
	}
	return false
}
```

Add these imports to the existing import block in `iter.go`:

```go
	"sort"
	"time"
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -run TestLoadDetailContent_IterCurrent -v`
Expected: PASS — both current-iter overlay tests green.

- [ ] **Step 5: Commit**

```bash
git add iter.go ledger_test.go
git commit -m "feat: add reviews and loop-context blocks to iter overlay"
```

---

## Task 12: Overlay guards — missing files and progress.md ID mismatch

**Files:**
- Modify: `ledger_test.go`

- [ ] **Step 1: Write tests for guards**

Append to `ledger_test.go`:

```go
func TestLoadDetailContent_IterCurrent_AllOverlayFilesMissing(t *testing.T) {
	tmpDir := t.TempDir()
	// No .ai/, no docs/, no overlay files at all.

	s := Sprint{
		ID: "0001", IDPrefix: "ITER", IsCurrent: true,
		Status: "in_progress", Section: "### ITER-0001 — Body",
	}
	got, err := LoadDetailContent(s, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(got, "ITER-0001") {
		t.Errorf("expected header line, got: %q", got[:30])
	}
	// "run status:" must be absent when iter-workflow-status.txt is missing.
	if strings.Contains(got, "run status:") {
		t.Errorf("expected no 'run status:' clause when file missing")
	}
	for _, header := range []string{"## Progress", "## Tasks", "## Reviews", "## Loop Context"} {
		if strings.Contains(got, header) {
			t.Errorf("expected %s to be omitted when source file missing", header)
		}
	}
	if !strings.Contains(got, "## Roadmap Section") {
		t.Errorf("expected Roadmap Section to still be present")
	}
}

func TestLoadDetailContent_IterCurrent_ProgressIDMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	os.MkdirAll(docsDir, 0755)
	// progress.md mentions a different ITER than the sprint we render.
	os.WriteFile(filepath.Join(docsDir, "progress.md"),
		[]byte("**Iteration:** ITER-0099\n**Tasks:** 0/0 done\n"), 0644)

	s := Sprint{
		ID: "0001", IDPrefix: "ITER", IsCurrent: true,
		Status: "in_progress", Section: "### ITER-0001 — Body",
	}
	got, err := LoadDetailContent(s, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "## Progress") {
		t.Errorf("expected Progress block to be skipped when progress.md references wrong ITER")
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test -run TestLoadDetailContent_IterCurrent -v`
Expected: PASS — guards already implemented in Tasks 10 and 11 (best-effort reads + ID-match check). If anything fails, fix the helper functions.

- [ ] **Step 3: Commit**

```bash
git add ledger_test.go
git commit -m "test: cover overlay guards for missing files and ID mismatch"
```

---

## Task 13: Watcher refactor — introduce layout enum, preserve existing behavior

**Files:**
- Modify: `watcher.go`
- Modify: `watcher_test.go`
- Modify: `model.go:75-80` (Init), `model.go:118` (after ledgerChangedMsg)

- [ ] **Step 1: Write the new test**

Append to `watcher_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestWatchProject_LedgerLayout_BehavesAsBefore -v`
Expected: FAIL — `watchProject` and `layoutLedger` undefined.

- [ ] **Step 3: Refactor watcher.go**

Replace `watcher.go` entirely with:

```go
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
		for _, d := range dirs {
			// Best-effort: missing directories are silently skipped.
			_ = watcher.Add(d)
		}
		if len(dirs) == 0 {
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
		if strings.Contains(event.Name, filepath.Join(".ai", "iter-run-temp")) {
			return strings.HasSuffix(base, ".md") || strings.HasSuffix(base, ".txt")
		}
		return false
	}
	return false
}
```

- [ ] **Step 4: Update model.go to use watchProject**

In `model.go`, modify the `model` struct (line 21) by adding a `lay layout` field:

```go
type model struct {
	sprints     []Sprint
	projectName string
	projectRoot string
	lay         layout
	table       table.Model
	viewport    viewport.Model
	focus       focus
	width       int
	height      int
	ready       bool
}
```

Modify `newModel` (line 33) to accept and store the layout:

```go
func newModel(sprints []Sprint, projectName, projectRoot string, lay layout) model {
```

In the function body, set `lay: lay,` in the struct literal returned at the bottom.

In `Init()` (line 75-80), replace:

```go
	return watchLedger(m.projectRoot)
```

with:

```go
	return watchProject(m.projectRoot, m.lay)
```

In `Update()`, find the `case ledgerChangedMsg:` block. The last return statement is currently:

```go
		return m, watchLedger(m.projectRoot)
```

Replace with:

```go
		return m, watchProject(m.projectRoot, m.lay)
```

Also fix the early-error return inside the same case (`return m, watchLedger(m.projectRoot)` on line 89): change to `return m, watchProject(m.projectRoot, m.lay)`.

- [ ] **Step 5: Update existing watcher tests to use new API**

In `watcher_test.go`, find every `cmd := watchLedger(...)` and replace with `cmd := watchProject(..., layoutLedger)`. The signatures of the two tests `TestWatchLedger_DetectsChange` and `TestWatchLedger_IgnoresOtherFiles` remain otherwise unchanged.

In `watcher_test.go`, find `m := newModel(sprints, "test", tmpDir)` in `TestModel_LedgerChangedMsg_ReloadsData` and replace with `m := newModel(sprints, "test", tmpDir, layoutLedger)`.

- [ ] **Step 6: Update main.go's newModel call**

In `main.go` line 56, replace:

```go
	m := newModel(sprints, projectName, projectRoot)
```

with:

```go
	m := newModel(sprints, projectName, projectRoot, layoutLedger)
```

(We'll wire up real layout detection in Task 15.)

- [ ] **Step 7: Run all tests**

Run: `go test ./...`
Expected: PASS — both the new ledger-layout test and all existing watcher/model tests green.

- [ ] **Step 8: Commit**

```bash
git add watcher.go watcher_test.go model.go main.go
git commit -m "refactor: layout-aware watcher (preserves ledger behavior)"
```

---

## Task 14: Watcher — iter-layout file detection and noise filtering

**Files:**
- Modify: `watcher_test.go`

- [ ] **Step 1: Write tests for iter-layout watching**

Append to `watcher_test.go`:

```go
func TestWatchProject_IterLayout_TriggersOnRoadmapChange(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(docsDir, 0755)
	os.MkdirAll(aiDir, 0755)
	roadmap := filepath.Join(docsDir, "roadmap.md")
	os.WriteFile(roadmap, []byte("### ITER-0001 — Hi\n"), 0644)

	cmd := watchProject(tmpDir, layoutIter)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	time.Sleep(200 * time.Millisecond)

	os.WriteFile(roadmap, []byte("### ITER-0001 — Hi\n### ITER-0002 — There\n"), 0644)

	select {
	case msg := <-done:
		if _, ok := msg.(ledgerChangedMsg); !ok {
			t.Errorf("expected ledgerChangedMsg, got %T", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for roadmap change")
	}
}

func TestWatchProject_IterLayout_TriggersOnCurrentIterChange(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	os.MkdirAll(aiDir, 0755)
	currentPath := filepath.Join(aiDir, "iter-current-iteration.txt")
	os.WriteFile(currentPath, []byte("ITER-0001"), 0644)

	cmd := watchProject(tmpDir, layoutIter)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	time.Sleep(200 * time.Millisecond)

	os.WriteFile(currentPath, []byte("ITER-0002"), 0644)

	select {
	case msg := <-done:
		if _, ok := msg.(ledgerChangedMsg); !ok {
			t.Errorf("expected ledgerChangedMsg, got %T", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for current-iter change")
	}
}

func TestWatchProject_IterLayout_IgnoresExtractTemp(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	extractDir := filepath.Join(aiDir, "iter-extract-temp")
	os.MkdirAll(extractDir, 0755)

	cmd := watchProject(tmpDir, layoutIter)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	time.Sleep(200 * time.Millisecond)

	os.WriteFile(filepath.Join(extractDir, "chunk_001.md"), []byte("noise"), 0644)

	select {
	case msg := <-done:
		t.Errorf("expected no message for iter-extract-temp file, got %T", msg)
	case <-time.After(500 * time.Millisecond):
		// Good — no message.
	}
}
```

- [ ] **Step 2: Run tests to verify them**

Run: `go test -run TestWatchProject_IterLayout -v`
Expected: PASS — all three tests green (the watcher already implements iter-layout watching from Task 13; we are verifying behavior here).

- [ ] **Step 3: Commit**

```bash
git add watcher_test.go
git commit -m "test: cover iter-layout watch triggers and extract-temp filtering"
```

---

## Task 15: DetectLayout + main.go wire-up

**Files:**
- Modify: `iter.go` (add DetectLayout)
- Modify: `main.go`
- Modify: `roadmap_test.go` (add DetectLayout tests)

- [ ] **Step 1: Write failing tests**

Append to `roadmap_test.go`:

```go
func TestDetectLayout_PrefersLedger(t *testing.T) {
	tmpDir := t.TempDir()
	aiDir := filepath.Join(tmpDir, ".ai")
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	os.MkdirAll(aiDir, 0755)
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(aiDir, "ledger.yaml"), []byte("sprints: []\n"), 0644)
	os.WriteFile(filepath.Join(docsDir, "roadmap.md"), []byte("### ITER-0001 — Hi\n"), 0644)

	if got := DetectLayout(tmpDir); got != layoutLedger {
		t.Errorf("expected layoutLedger, got %v", got)
	}
}

func TestDetectLayout_FallsBackToIter(t *testing.T) {
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "iterations")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "roadmap.md"), []byte("### ITER-0001 — Hi\n"), 0644)

	if got := DetectLayout(tmpDir); got != layoutIter {
		t.Errorf("expected layoutIter, got %v", got)
	}
}

func TestDetectLayout_DefaultsToLedgerWhenNothingFound(t *testing.T) {
	tmpDir := t.TempDir()
	// Defaults to layoutLedger so LoadSprints surfaces the existing
	// "no ledger found" error verbatim — no behavior change for empty projects.
	if got := DetectLayout(tmpDir); got != layoutLedger {
		t.Errorf("expected layoutLedger default, got %v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestDetectLayout -v`
Expected: FAIL — `DetectLayout` undefined.

- [ ] **Step 3: Add DetectLayout to iter.go**

Append to `iter.go`:

```go
// DetectLayout returns which layout LoadSprints will use for the given project.
// Selection order matches LoadSprints exactly. Defaults to layoutLedger when
// nothing is present so the existing "no ledger found" error path is preserved.
func DetectLayout(projectRoot string) layout {
	if _, err := os.Stat(filepath.Join(projectRoot, ".ai", "ledger.yaml")); err == nil {
		return layoutLedger
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".ai", "ledger.tsv")); err == nil {
		return layoutLedger
	}
	for _, rel := range roadmapSearchOrder {
		if _, err := os.Stat(filepath.Join(projectRoot, rel)); err == nil {
			return layoutIter
		}
	}
	return layoutLedger
}
```

- [ ] **Step 4: Wire DetectLayout into main.go**

In `main.go`, between the `LoadSprints` call and the `newModel` call, add a `DetectLayout` call. Replace lines 49-56:

```go
	sprints, err := LoadSprints(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectName := filepath.Base(projectRoot)
	m := newModel(sprints, projectName, projectRoot)
```

with:

```go
	sprints, err := LoadSprints(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	lay := DetectLayout(projectRoot)
	projectName := filepath.Base(projectRoot)
	m := newModel(sprints, projectName, projectRoot, lay)
```

- [ ] **Step 5: Run all tests**

Run: `go test ./...`
Expected: PASS — DetectLayout tests green, everything else still green.

- [ ] **Step 6: Build and verify the binary**

Run: `go build -o sprint-ledger .`
Expected: clean build, no errors.

- [ ] **Step 7: Commit**

```bash
git add iter.go main.go roadmap_test.go
git commit -m "feat: detect layout at startup and wire into watcher"
```

---

## Task 16: Real-project smoke verification

**Files:** none (manual verification step)

- [ ] **Step 1: Run against kanmail**

Run: `./sprint-ledger /Users/harper/Public/src/2389/kanmail`
Expected:
- Table populated with ITER-0000 through ITER-0010 (and sub-iters like ITER-0003a, ITER-0006a-i)
- Status column shows colored icons (mostly done for early iters, pending for late ones, in_progress for ITER-0002, the one with the complex status string)
- Cursor on the current iteration (ITER-0004a per `iter-current-iteration.txt`) opens a detail view with:
  - Header: `ITER-0004a [pending | run status: fail]` (or whatever the live status is)
  - `## Progress`, `## Tasks`, `## Reviews`, `## Loop Context`, `## Roadmap Section` blocks in that order
- Non-current iterations open with just the roadmap section body
- Press `r` to manually reload; table refreshes

Press `q` to quit.

- [ ] **Step 2: Run against door**

Run: `./sprint-ledger /Users/harper/Public/src/2389/door`
Expected:
- Table populated with ITER-0000 through ITER-0011 (and sub-iters ITER-0003a, ITER-0003b)
- ITER-0005 is the current iteration (per `iter-current-iteration.txt`). Its explicit roadmap status is `pending`, so it shows `pending` in the table; the overlay header in its detail pane shows `ITER-0005 [pending | run status: ...]` with both signals.
- The Progress block is OMITTED for ITER-0005 because `docs/iterations/progress.md` references `ITER-0004` (stale).
- Other iterations open cleanly with their roadmap sections.

Press `q` to quit.

- [ ] **Step 3: Verify ledger projects still work**

Run: `./sprint-ledger .` (from this project's root — sprint-ledger-browser uses no `.ai/`, so this should error gracefully)

Expected: error message about no ledger found. No crash.

Then run against any project with a real `.ai/ledger.yaml` or `.ai/ledger.tsv` and confirm existing behavior is unchanged.

- [ ] **Step 4: Run full test suite one more time**

Run: `go test ./... -count=1`
Expected: PASS — full test suite green.

- [ ] **Step 5: Commit (only if any small fixes were needed during smoke testing)**

If smoke testing surfaced bugs, fix them with their own focused commits. Otherwise nothing to commit here.

---

## Done

Push the branch and open a PR:

```bash
git push -u origin feat/iter-layout-support
gh pr create --title "feat: support iter-* layout for kanmail and door" --body "$(cat <<'EOF'
## Summary
- Add support for the iter-* / Ralph-style project layout (no ledger file; iterations parsed from `docs/iterations/roadmap.md` with rich live overlay)
- Selection order: `.ai/ledger.yaml` → `.ai/ledger.tsv` → `docs/iterations/roadmap.md` → `.ai/iter-scope-temp/iterations.md`
- For the current iteration (per `.ai/iter-current-iteration.txt`), the detail pane prepends a status-first overlay: header → progress → tasks → reviews → loop context → roadmap section
- Status normalization: `done`→completed, `fail`→failed, `pending`→pending, `in_progress (...)` → in_progress; missing Status lines inferred from position relative to the current iteration
- Watcher is now layout-aware; iter-layout watches roadmap + key state files, ignores noisy `iter-extract-temp/` and `iter-audit-temp/`

## Test plan
- [x] `go test ./...` green
- [x] `./sprint-ledger /Users/harper/Public/src/2389/kanmail` renders ITER table and overlay
- [x] `./sprint-ledger /Users/harper/Public/src/2389/door` renders ITER table and overlay
- [x] Existing ledger projects unchanged

Spec: `docs/superpowers/specs/2026-05-19-iter-layout-support-design.md`
Plan: `docs/superpowers/plans/2026-05-19-iter-layout-support.md`
EOF
)"
```
