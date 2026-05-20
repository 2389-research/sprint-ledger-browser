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
