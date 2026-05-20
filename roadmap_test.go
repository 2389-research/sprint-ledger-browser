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
