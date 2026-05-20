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
