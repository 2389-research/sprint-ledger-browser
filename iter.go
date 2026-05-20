// ABOUTME: Loader and detail composer for the iter-* project layout.
// ABOUTME: Reads docs/iterations/roadmap.md plus .ai/iter-* state files.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// roadmapSearchOrder is the fixed list of relative paths LoadIterRoadmap tries.
var roadmapSearchOrder = []string{
	filepath.Join("docs", "iterations", "roadmap.md"),
	filepath.Join(".ai", "iter-scope-temp", "iterations.md"),
}

// LoadIterRoadmap finds and parses a roadmap, then performs the current-iter
// detection and status-inference pass. Returns an error if no roadmap is found.
func LoadIterRoadmap(projectRoot string) ([]Sprint, error) {
	var (
		sprints []Sprint
		err     error
		found   bool
	)
	for _, rel := range roadmapSearchOrder {
		path := filepath.Join(projectRoot, rel)
		f, openErr := os.Open(path)
		if openErr != nil {
			continue
		}
		sprints, err = ParseRoadmap(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		found = true
		break
	}
	if !found {
		return nil, fmt.Errorf("no roadmap found (looked at %v relative to %s)", roadmapSearchOrder, projectRoot)
	}

	currentID := readCurrentIterationID(projectRoot)
	applyCurrentAndInference(sprints, currentID)
	return sprints, nil
}

// readCurrentIterationID reads .ai/iter-current-iteration.txt and strips the
// "ITER-" prefix. Returns "" if the file is missing or empty.
func readCurrentIterationID(projectRoot string) string {
	data, err := os.ReadFile(filepath.Join(projectRoot, ".ai", "iter-current-iteration.txt"))
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(string(data))
	return strings.TrimPrefix(raw, "ITER-")
}

// applyCurrentAndInference mutates sprints in place:
//   - sets IsCurrent on the sprint whose ID matches currentID
//   - fills empty Status fields based on position relative to the current iter
//
// If currentID is empty (no marker file) and Status is empty, leaves it "unknown".
func applyCurrentAndInference(sprints []Sprint, currentID string) {
	currentIdx := -1
	for i := range sprints {
		if sprints[i].ID == currentID {
			currentIdx = i
			sprints[i].IsCurrent = true
		}
	}
	for i := range sprints {
		if sprints[i].Status != "" {
			continue
		}
		switch {
		case currentIdx < 0:
			sprints[i].Status = "unknown"
		case i < currentIdx:
			sprints[i].Status = "completed"
		case i == currentIdx:
			sprints[i].Status = "in_progress"
		default:
			sprints[i].Status = "pending"
		}
	}
}
