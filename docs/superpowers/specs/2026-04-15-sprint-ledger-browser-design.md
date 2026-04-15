# Sprint Ledger Browser — Design Spec

## Overview

A read-only TUI for viewing `.ai/ledger.tsv` files produced by the `spec_to_sprints` pipeline. Displays sprint status in a navigable table with a detail pane showing the corresponding `SPRINT-*.md` content.

## CLI Interface

```
sprint-ledger <project-root>
```

- `<project-root>` is required. The tool reads `<project-root>/.ai/ledger.tsv` and `<project-root>/.ai/sprints/SPRINT-*.md`.
- If ledger file is missing, print an error and exit with code 1.
- No flags, no subcommands.

## Data Model

### Input: `ledger.tsv`

Tab-separated, 5 columns with a header row:

| Column       | Type             | Description                        |
|------------- |------------------|------------------------------------|
| `sprint_id`  | String (e.g. `001`) | Sprint identifier                 |
| `title`      | String           | Sprint title                       |
| `status`     | Enum             | `planned`, `in_progress`, `completed`/`complete`, `skipped` |
| `created_at` | ISO 8601 UTC     | When sprint was created            |
| `updated_at` | ISO 8601 UTC     | When sprint was last updated       |

### Input: `SPRINT-*.md`

Markdown files in `.ai/sprints/`. Matched to ledger rows by sprint_id: sprint `001` maps to `SPRINT-001.md`. Content is displayed as plain text in the detail viewport (no markdown rendering).

### Go Struct

```go
type Sprint struct {
    ID        string
    Title     string
    Status    string
    CreatedAt time.Time
    UpdatedAt time.Time
    DetailPath string // path to SPRINT-*.md, empty if file not found
}
```

## Architecture

Single Bubbletea program with a root model containing two child models:

1. **SprintTable** — top pane, navigable table of sprints
2. **DetailViewport** — bottom pane, scrollable text content of the selected sprint's markdown file

### Root Model Responsibilities

- Manage terminal resize events and allocate pane heights (table ~40%, detail ~60%)
- Route key events to the focused pane
- Track which pane has focus (table or detail)
- Load sprint data on init

### SprintTable

- Uses `bubbles/table` component
- Columns: ID, Title, Status (with icon), Updated (short date)
- Row highlight follows cursor
- Emits a message when the selected row changes so the root model can update the detail pane

### DetailViewport

- Uses `bubbles/viewport` component
- Displays the raw text content of the `SPRINT-*.md` file matching the highlighted sprint
- If no matching file exists, shows "No sprint detail file found for SPRINT-{id}.md"
- Content loaded lazily — read file from disk when selection changes

## UI Layout

```
┌─────────────────────────────────────────────┐
│  sprint-ledger: <project-name>              │
├────┬──────────────────────┬────────┬────────┤
│ ID │ Title                │ Status │ Updated│
├────┼──────────────────────┼────────┼────────┤
│001 │ Project Scaffolding  │ ● done │ Apr 13 │
│002 │ Persistence Layer    │ ● done │ Apr 13 │
│003▸│ Slider Component     │ ◐ wip  │ Apr 14 │
│004 │ Task History         │ ○ plan │ Apr 13 │
├────┴──────────────────────┴────────┴────────┤
│ # Sprint 003 — Slider Component & Task Form │
│                                             │
│ ## Scope                                    │
│ - Build energy slider (0-10)                │
│ - Create task logging form...               │
│                                             │
│                              ↕ scroll  q quit│
└─────────────────────────────────────────────┘
```

### Project Name

Derived from the basename of the project root argument. Displayed in the header.

### Status Display

| Status        | Icon | Color  |
|---------------|------|--------|
| `completed`   | `●`  | Green  |
| `in_progress` | `◐`  | Yellow |
| `planned`     | `○`  | Dim    |
| `skipped`     | `⊘`  | Dim + strikethrough |

### Timestamp Display

`updated_at` formatted as short date: `Apr 13`, `Jan 02`, etc. using Go's `time.Format("Jan 02")`.

### Pane Sizing

- Header: 1 line
- Table: ~40% of remaining height
- Detail: ~60% of remaining height
- Footer help text: 1 line
- Responds to terminal resize events

## Key Bindings

| Key              | Context | Action                          |
|------------------|---------|---------------------------------|
| `j` / `↓`       | Table   | Move to next sprint             |
| `k` / `↑`       | Table   | Move to previous sprint         |
| `enter` / `tab`  | Any     | Toggle focus between table and detail pane |
| `j` / `↓`       | Detail  | Scroll detail down              |
| `k` / `↑`       | Detail  | Scroll detail up                |
| `q` / `ctrl+c`  | Any     | Quit                            |

Focus indicator: the active pane's border is highlighted; the inactive pane's border is dimmed.

## Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/lipgloss` — styling and layout
- `github.com/charmbracelet/bubbles` — table and viewport components

Standard library only for TSV parsing (`encoding/csv` with tab delimiter), file I/O, and time parsing.

## Error Handling

- Missing `<project-root>/.ai/ledger.tsv`: print error message to stderr, exit 1
- Malformed TSV rows (wrong column count): skip the row, continue parsing
- Status normalization: treat `complete` and `completed` as equivalent (both display as completed)
- Missing `SPRINT-*.md` for a sprint: show placeholder text in detail pane
- Empty ledger (header only): show "No sprints found" message

## What This Is Not

- No file watching or live reload
- No editing or status mutation
- No multi-project aggregation
- No network access
- No markdown rendering (glamour, etc.) — plain text display of markdown source
