// ABOUTME: Unit tests for TSV and YAML ledger parsing and sprint detail file resolution.
// ABOUTME: Covers happy path, malformed input, status normalization, and missing files.
package main

import (
	"os"
	"path/filepath"
	"strings"
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

	if len(sprints) != 2 {
		t.Fatalf("expected 2 sprints (skipping malformed), got %d", len(sprints))
	}

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
	content, err := LoadDetailContent(s, "")
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
	content, err := LoadDetailContent(s, "")
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
	_, err := LoadDetailContent(s, "")
	if err == nil {
		t.Error("expected error for unreadable file")
	}
}

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

func TestParseYAMLLedger_ValidFile(t *testing.T) {
	data, err := os.Open(filepath.Join("testdata", "yaml", ".ai", "ledger.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	sprints, err := ParseYAMLLedger(data)
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
	if s.Title != "Foundation & Data Model" {
		t.Errorf("expected title 'Foundation & Data Model', got '%s'", s.Title)
	}
	if s.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", s.Status)
	}
	expected := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	if !s.UpdatedAt.Equal(expected) {
		t.Errorf("expected UpdatedAt %v, got %v", expected, s.UpdatedAt)
	}
}

func TestParseYAMLLedger_StatusNormalization(t *testing.T) {
	yaml := `
sprints:
  - id: '001'
    title: Test
    status: complete
    created_at: '2026-04-10T19:11:51Z'
    updated_at: '2026-04-10T19:11:51Z'
`
	sprints, err := ParseYAMLLedger(strings.NewReader(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 1 {
		t.Fatalf("expected 1 sprint, got %d", len(sprints))
	}
	if sprints[0].Status != "completed" {
		t.Errorf("expected 'completed' (normalized), got '%s'", sprints[0].Status)
	}
}

func TestParseYAMLLedger_EmptySprints(t *testing.T) {
	yaml := `
project:
  name: Empty
sprints: []
`
	sprints, err := ParseYAMLLedger(strings.NewReader(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(sprints) != 0 {
		t.Fatalf("expected 0 sprints, got %d", len(sprints))
	}
}

func TestParseYAMLLedger_InvalidYAML(t *testing.T) {
	bad := strings.NewReader("{{not valid yaml at all")
	_, err := ParseYAMLLedger(bad)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadSprints_YAMLProject(t *testing.T) {
	sprints, err := LoadSprints(filepath.Join("testdata", "yaml"))
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
	// outside the project directory
	if sprints[0].DetailPath != "" {
		resolved, _ := filepath.Abs(sprints[0].DetailPath)
		projectAbs, _ := filepath.Abs(tmpDir)
		if !strings.HasPrefix(resolved, projectAbs) {
			t.Errorf("DetailPath escaped project root: %s", resolved)
		}
	}
}

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
