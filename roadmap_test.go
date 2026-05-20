// ABOUTME: Unit tests for the roadmap.md iteration parser.
// ABOUTME: Covers heading detection, status normalization, and section preservation.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
