// ABOUTME: Loader and detail composer for the iter-* project layout.
// ABOUTME: Reads docs/iterations/roadmap.md plus .ai/iter-* state files.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoRoadmap is returned by LoadIterRoadmap when no roadmap file is found
// at any of the search paths. It distinguishes "missing file" from "file exists
// but failed to parse", so callers can choose to fall through or surface the error.
var ErrNoRoadmap = errors.New("no roadmap found")

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
		return nil, fmt.Errorf("%w (looked at %v relative to %s)", ErrNoRoadmap, roadmapSearchOrder, projectRoot)
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

// renderIterDetailWithOverlay composes the status-first overlay for the
// current iteration: header + progress + tasks + reviews + loop context +
// roadmap section. Each block is best-effort; missing files are silently
// omitted.
func renderIterDetailWithOverlay(s Sprint, projectRoot string) string {
	var b strings.Builder
	prefix := effectivePrefix(s)

	// Header: "ITER-0003a  [in_progress | run status: fail]"
	b.WriteString(prefix)
	b.WriteString("-")
	b.WriteString(s.ID)
	b.WriteString("  [")
	b.WriteString(s.Status)
	if runStatus := readTrimmed(filepath.Join(projectRoot, ".ai", "iter-workflow-status.txt")); runStatus != "" {
		b.WriteString(" | run status: ")
		b.WriteString(runStatus)
	}
	b.WriteString("]\n\n")

	// Progress block (docs/iterations/progress.md) — only if it references the current ITER ID.
	progressPath := filepath.Join(projectRoot, "docs", "iterations", "progress.md")
	if progress, ok := readProgressIfMatches(progressPath, prefix+"-"+s.ID); ok {
		b.WriteString("## Progress\n\n")
		b.WriteString(progress)
		b.WriteString("\n\n")
	}

	// Tasks block (.ai/iter-run-temp/task_status.txt)
	if tasks := readTrimmed(filepath.Join(projectRoot, ".ai", "iter-run-temp", "task_status.txt")); tasks != "" {
		b.WriteString("## Tasks\n\n")
		b.WriteString(tasks)
		b.WriteString("\n\n")
	}

	// Roadmap section last.
	b.WriteString("## Roadmap Section\n\n")
	b.WriteString(s.Section)
	return b.String()
}

// readTrimmed reads a file and returns its trimmed contents, or "" on any error.
func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// readProgressIfMatches returns the file contents only when the first ~20 lines
// reference the given fullID (e.g. "ITER-0003a"). Prevents stale snapshots
// showing the wrong iteration.
func readProgressIfMatches(path, fullID string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	content := string(data)
	head := content
	if nl := indexNthByte(content, '\n', 20); nl > 0 {
		head = content[:nl]
	}
	if !strings.Contains(head, fullID) {
		return "", false
	}
	return strings.TrimSpace(content), true
}

// indexNthByte returns the byte offset of the n-th occurrence of c, or -1 if
// it occurs fewer than n times.
func indexNthByte(s string, c byte, n int) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
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
