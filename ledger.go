// ABOUTME: Sprint data model and TSV ledger parser.
// ABOUTME: Reads .ai/ledger.tsv files and resolves paths to SPRINT-*.md detail files.
package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
// Skips the header row. Uses header to locate columns by name, so ledgers
// with extra columns (depends_on, frs, etc.) are handled correctly.
// Normalizes "complete" to "completed".
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
	if err != nil {
		return nil, fmt.Errorf("reading TSV: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	// Build column index from header row
	colIdx := map[string]int{}
	for i, name := range records[0] {
		colIdx[name] = i
	}

	// Require at minimum: sprint_id, title, status
	idCol, hasID := colIdx["sprint_id"]
	titleCol, hasTitle := colIdx["title"]
	statusCol, hasStatus := colIdx["status"]
	if !hasID || !hasTitle || !hasStatus {
		return nil, fmt.Errorf("ledger missing required columns (need sprint_id, title, status)")
	}

	createdCol := colIdx["created_at"] // 0 if missing, but we check existence
	updatedCol := colIdx["updated_at"]
	_, hasCreated := colIdx["created_at"]
	_, hasUpdated := colIdx["updated_at"]

	var sprints []Sprint
	for _, row := range records[1:] {
		// Skip rows that don't have enough columns for the required fields
		maxRequired := max(idCol, titleCol, statusCol)
		if len(row) <= maxRequired {
			continue
		}

		status := row[statusCol]
		if status == "complete" {
			status = "completed"
		}

		var createdAt, updatedAt time.Time
		if hasCreated && createdCol < len(row) {
			createdAt, _ = time.Parse(time.RFC3339, row[createdCol])
		}
		if hasUpdated && updatedCol < len(row) {
			updatedAt, _ = time.Parse(time.RFC3339, row[updatedCol])
		}

		sprints = append(sprints, Sprint{
			ID:        row[idCol],
			Title:     row[titleCol],
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
