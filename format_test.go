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
		{"failed", "✗"},
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
