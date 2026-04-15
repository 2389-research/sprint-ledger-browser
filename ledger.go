// ABOUTME: Sprint data model and TSV ledger parser.
// ABOUTME: Reads .ai/ledger.tsv files and resolves paths to SPRINT-*.md detail files.
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
// Skips the header row. Skips rows with wrong column count.
// Normalizes "complete" to "completed".
func ParseLedger(r io.Reader) ([]Sprint, error) {
	reader := csv.NewReader(r)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading TSV: %w", err)
	}

	var sprints []Sprint
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) != 5 {
			continue
		}

		status := row[2]
		if status == "complete" {
			status = "completed"
		}

		createdAt, err := time.Parse(time.RFC3339, row[3])
		if err != nil {
			createdAt = time.Time{}
		}
		updatedAt, err := time.Parse(time.RFC3339, row[4])
		if err != nil {
			updatedAt = time.Time{}
		}

		sprints = append(sprints, Sprint{
			ID:        row[0],
			Title:     row[1],
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

	for i := range sprints {
		detailPath := filepath.Join(projectRoot, ".ai", "sprints", fmt.Sprintf("SPRINT-%s.md", sprints[i].ID))
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
