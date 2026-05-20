// ABOUTME: Parser for iteration roadmap markdown files.
// ABOUTME: Reads docs/iterations/roadmap.md style files and returns iteration sprints.
package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// iterHeadingRE matches "### ITER-<id> <dash> <title>".
// Accepts em-dash (—), en-dash (–), and ASCII hyphen (-).
var iterHeadingRE = regexp.MustCompile(`^### ITER-(\S+)\s+[\x{2014}\x{2013}\-]\s+(.*)$`)

// statusLineRE captures the value after "**Status:**".
var statusLineRE = regexp.MustCompile(`^\*\*Status:\*\*\s*(.*)$`)

// ParseRoadmap reads a roadmap markdown file and returns one Sprint per
// "### ITER-..." heading, each carrying its raw section text in Section.
// Status normalization and current-iteration inference happen in LoadSprints,
// not here.
func ParseRoadmap(r io.Reader) ([]Sprint, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var sprints []Sprint
	var current *Sprint
	var section strings.Builder

	flush := func() {
		if current == nil {
			return
		}
		current.Section = strings.TrimRight(section.String(), "\n")
		sprints = append(sprints, *current)
		current = nil
		section.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()

		if m := iterHeadingRE.FindStringSubmatch(line); m != nil {
			flush()
			current = &Sprint{
				ID:       m[1],
				IDPrefix: "ITER",
				Title:    strings.TrimSpace(m[2]),
			}
			section.WriteString(line)
			section.WriteString("\n")
			continue
		}

		// A different ### heading ends the current section.
		if strings.HasPrefix(line, "### ") && current != nil {
			flush()
			continue
		}

		if current != nil && current.Status == "" {
			if m := statusLineRE.FindStringSubmatch(line); m != nil {
				current.Status = normalizeStatus(m[1])
			}
		}

		if current != nil {
			section.WriteString(line)
			section.WriteString("\n")
		}
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning roadmap: %w", err)
	}
	return sprints, nil
}

// normalizeStatus extracts the leading token from a raw status string and maps
// it to a canonical status. Examples:
//
//	"done"                            -> "completed"
//	"done:ITER-0001"                  -> "completed"
//	"in_progress (ITER-0002a done…)"  -> "in_progress"
//	"fail"                            -> "failed"
//	"pending"                         -> "pending"
//	anything else                     -> "unknown"
func normalizeStatus(raw string) string {
	raw = strings.TrimSpace(raw)
	// Leading token: stop at first space, "(", or ":".
	idx := strings.IndexAny(raw, " (:")
	token := raw
	if idx >= 0 {
		token = raw[:idx]
	}
	switch token {
	case "done":
		return "completed"
	case "pending":
		return "pending"
	case "in_progress":
		return "in_progress"
	case "fail":
		return "failed"
	default:
		return "unknown"
	}
}
