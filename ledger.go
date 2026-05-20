// ABOUTME: Sprint loader — picks YAML/TSV ledger or iter-* roadmap and returns Sprint records.
// ABOUTME: Resolves detail-file paths for ledger sprints; iter sprints carry Section content inline.
package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

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

// yamlLedger is the top-level structure of a YAML ledger file.
type yamlLedger struct {
	Sprints []yamlSprint `yaml:"sprints"`
}

// yamlSprint maps a single sprint entry in the YAML format.
type yamlSprint struct {
	ID        string `yaml:"id"`
	Title     string `yaml:"title"`
	Status    string `yaml:"status"`
	CreatedAt string `yaml:"created_at"`
	UpdatedAt string `yaml:"updated_at"`
}

// ParseYAMLLedger reads a YAML ledger file and returns Sprint records.
// Normalizes "complete" to "completed".
func ParseYAMLLedger(r io.Reader) ([]Sprint, error) {
	var ledger yamlLedger
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&ledger); err != nil {
		return nil, fmt.Errorf("parsing YAML ledger: %w", err)
	}

	var sprints []Sprint
	for _, ys := range ledger.Sprints {
		status := ys.Status
		if status == "complete" {
			status = "completed"
		}

		var createdAt, updatedAt time.Time
		if ys.CreatedAt != "" {
			createdAt, _ = time.Parse(time.RFC3339, ys.CreatedAt)
		}
		if ys.UpdatedAt != "" {
			updatedAt, _ = time.Parse(time.RFC3339, ys.UpdatedAt)
		}

		sprints = append(sprints, Sprint{
			ID:        ys.ID,
			Title:     ys.Title,
			Status:    status,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	return sprints, nil
}

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
	sprints, err := LoadIterRoadmap(projectRoot)
	if err == nil {
		return sprints, nil
	}
	if !errors.Is(err, ErrNoRoadmap) {
		return nil, err // real parse error, surface it
	}

	return nil, fmt.Errorf("opening ledger: no ledger.yaml, ledger.tsv, or roadmap.md found for %s", projectRoot)
}

// resolveDetailPaths populates DetailPath for ledger-layout sprints.
func resolveDetailPaths(sprints []Sprint, projectRoot string) {
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
}

// effectivePrefix returns the ID prefix for display, defaulting to "SPRINT"
// when Sprint.IDPrefix is empty (back-compat for ledger-layout sprints).
func effectivePrefix(s Sprint) string {
	if s.IDPrefix == "" {
		return "SPRINT"
	}
	return s.IDPrefix
}

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
