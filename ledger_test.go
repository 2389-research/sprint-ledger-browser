// ABOUTME: Unit tests for TSV ledger parsing and sprint detail file resolution.
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
