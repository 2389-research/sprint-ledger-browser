# iter-* layout support for sprint-ledger-browser

**Status:** Design approved
**Date:** 2026-05-19
**Author:** Claude (with Doctor Biz)

## Problem

`sprint-ledger-browser` today reads only one shape of `.ai/` directory:

- `.ai/ledger.{yaml,tsv}` — a flat list of sprints with `id`, `title`, `status`, dates
- `.ai/sprints/SPRINT-{id}.md` — one markdown detail file per sprint

Two other 2389 projects, `kanmail` and `door`, use a different layout driven by the iteration-loop ("iter-*") pipeline. They have no ledger file; the sprint list lives implicitly in a roadmap markdown, and per-iteration state is spread across several `iter-*` files in `.ai/` and `docs/iterations/`.

We want to support both layouts in one binary so the same tool works across all three projects without configuration.

## Source layouts to support

### Ledger layout (existing — unchanged)

```
.ai/
├── ledger.yaml          # or ledger.tsv
└── sprints/
    └── SPRINT-{id}.md
```

### Iter-* layout (new)

```
docs/iterations/
├── roadmap.md           # primary source for the iteration list
├── iteration-log.md     # (not read; for reference)
└── progress.md          # current-iter progress snapshot

.ai/
├── iter-current-iteration.txt    # which iter is running now (e.g. "ITER-0004a")
├── iter-workflow-status.txt      # outcome of current run (e.g. "fail")
├── iter-loop-context.txt         # narrative on what just happened
├── iter-scope-temp/
│   └── iterations.md             # fallback roadmap location
└── iter-run-temp/
    ├── task_status.txt           # "TASK-NN|status" per line
    ├── tasks.md, progress.md     # live state
    └── *_review_*.md             # scope/spec/quality review artifacts
```

#### Roadmap markdown format

Each iteration is a top-level `### ITER-...` section:

```markdown
### ITER-0001 — Foundation Hardening
**Status:** done
**Stories:** STORY-0276, STORY-0277, ...
**Rationale:** ...

### ITER-0002 — Auth, Tenancy, RBAC, ...
**Status:** in_progress (ITER-0002a done:2026-05-15 — ingestion; ITER-0002b pending — ...)
...
```

Sub-iteration headings (`### ITER-0003a`, `### ITER-0006a-i`) also appear as top-level sections when they exist independently. When a parent has sub-iters tracked only in its status string, the parent gets the row and the sub-iters appear inside the section text.

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Roadmap discovery | Fixed search order: `docs/iterations/roadmap.md` → `.ai/iter-scope-temp/iterations.md` | Both projects already use the canonical path; fallback covers in-progress build artifacts. |
| Detail richness | Section + live overlay | The whole point is rich UX for the current iteration. |
| Overlay order | Status-first: header → progress → tasks → reviews → loop-context → roadmap section | Most actionable info at top. |
| Status normalization | First token before space/paren/colon; map `done`→`completed`, `fail`→`failed`, others pass through | Handles compound strings like `done:ITER-0001` and `in_progress (...)`. |
| Missing status inference | Use `iter-current-iteration.txt` + roadmap order: before current → `completed`, current → `in_progress`, after → `pending`, no current marker → `unknown` | Door's `iterations.md` has no Status lines; this preserves usefulness. |
| Architecture | Inline branch (Approach 1): extend `Sprint` struct, add `roadmap.go`, third branch in `LoadSprints` | Minimal churn; matches single-binary ethos. |
| Reviews in overlay | Filename + mtime, not content | Door has 6+ review files per run; inlining buries everything else. |

## Architecture

### Data model

`Sprint` gains three optional fields:

```go
type Sprint struct {
    ID         string     // bare ID, e.g. "0001a"
    IDPrefix   string     // "SPRINT" (default/empty) or "ITER"
    Title      string
    Status     string     // completed | in_progress | failed | pending | unknown
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DetailPath string     // populated for ledger layout
    Section    string     // populated for iter-* layout — roadmap subsection
    IsCurrent  bool       // matches .ai/iter-current-iteration.txt
}
```

`IDPrefix` is empty for ledger sprints (back-compat: callers that read the field must treat empty as `"SPRINT"`). Iter sprints set it to `"ITER"`. Detail-view headers render `effectivePrefix(s) + "-" + s.ID`, where `effectivePrefix` is a one-line helper that returns `"SPRINT"` when `s.IDPrefix` is empty and `s.IDPrefix` otherwise. The list view's ID column stays narrow and shows only the bare ID.

### Selection in `LoadSprints`

```
1. .ai/ledger.yaml          → existing yaml parser
2. .ai/ledger.tsv           → existing tsv parser
3. docs/iterations/roadmap.md         → iter-* loader
4. .ai/iter-scope-temp/iterations.md  → iter-* loader (fallback)
5. none → existing error
```

### Roadmap parser (new file: `roadmap.go`)

```go
func ParseRoadmap(r io.Reader) ([]Sprint, error)
```

- Section starts at a line matching `^### ITER-(\S+)\s+[—–-]\s+(.+)$` and runs until the next `^### ` or EOF.
- ID = capture group 1. Title = capture group 2.
- First line matching `^\*\*Status:\*\*\s*(.*)$` inside the section captures the raw status string. Then extract the leading token: scan the captured string from the start and stop at the first ` `, `(`, or `:`. Examples: `done` → `done`; `done:ITER-0001` → `done`; `in_progress (ITER-0002a done — ...)` → `in_progress`.
- Status mapping: `done`→`completed`, `pending`→`pending`, `in_progress`→`in_progress`, `fail`→`failed`, anything else → `unknown` (log the unmatched token).
- No Status line → leave `Status = ""` for the inference pass.
- `Section` = full section text including heading. Used verbatim for non-current iterations.
- Not parsed (preserved in Section text): Stories, Rationale, Impacted scenarios, etc.
- Created/updated timestamps: not present in roadmap format; both zero-valued. The "Updated" column shows blank for iter-* sprints.
- Dash variants `—`, `–`, `-` all accepted.

### Inference pass (in `LoadSprints` after `ParseRoadmap`)

```
currentID := strings.TrimSpace(read .ai/iter-current-iteration.txt)
currentIdx := index in sprints where sprint matches currentID (-1 if not found)

for i, s in sprints:
    if s.ID matches currentID: s.IsCurrent = true
    if s.Status == "":
        if currentIdx < 0:      s.Status = "unknown"
        elif i < currentIdx:    s.Status = "completed"
        elif i == currentIdx:   s.Status = "in_progress"
        else:                   s.Status = "pending"
```

Order = roadmap section order. Explicit `**Status:**` lines always win over inference. Sub-iterations mentioned only inside another section's status string do not become separate rows.

### Detail composer

Signature change:

```go
func LoadDetailContent(s Sprint, projectRoot string) (string, error)
```

(All current call sites in `model.go` already have `projectRoot` in scope.)

Branch logic:

```
if s.Section != "":                      // iter-* layout
    if s.IsCurrent:
        return renderIterDetailWithOverlay(s, projectRoot)
    return s.Section
else:                                    // ledger layout
    existing logic (read s.DetailPath)
```

`renderIterDetailWithOverlay` produces:

```
ITER-0004a  [in_progress | run status: fail]

## Progress
<docs/iterations/progress.md, if its content references the current ITER>

## Tasks
<.ai/iter-run-temp/task_status.txt>

## Reviews
<filename + mtime per file matching .ai/iter-run-temp/{*_review_*.md, fresh_spec_*.md, scope.md}>

## Loop Context
<.ai/iter-loop-context.txt>

## Roadmap Section
<s.Section verbatim>
```

Sanity guards:
- Each helper read is best-effort. Missing or unreadable files → that block is omitted, no error.
- `progress.md` is included only if its first ~20 lines contain the current ITER-ID; prevents stale snapshots showing the wrong iteration.
- `run status:` clause in the header is dropped when `iter-workflow-status.txt` is missing or empty.
- Non-current iterations return `s.Section` verbatim — no file reads.

### Watcher extension

`watcher.go`: `watchLedger` becomes `watchProject` and takes a layout flag.

```go
type layout int
const (
    layoutLedger layout = iota
    layoutIter
)

func watchProject(projectRoot string, lay layout) tea.Cmd
```

- `layoutLedger`: watch `.ai/` for changes to `ledger.yaml` / `ledger.tsv` (unchanged behavior).
- `layoutIter`: watch a small explicit list:
  - `docs/iterations/roadmap.md`
  - `.ai/iter-scope-temp/iterations.md`
  - `.ai/iter-current-iteration.txt`
  - `.ai/iter-workflow-status.txt`
  - `.ai/iter-loop-context.txt`
  - `.ai/iter-run-temp/` directory
  - `docs/iterations/progress.md`
- Filter inside `.ai/iter-run-temp/` to `.md` and `.txt` only — ignores `.run_scope_rounds` and other counter files.
- Do NOT watch `.ai/iter-extract-temp/` or `.ai/iter-audit-temp/` — too noisy, irrelevant.
- Debounce: 100ms single timer (same as current).

Layout is detected at startup by a new `DetectLayout(projectRoot)` helper running the same priority order as `LoadSprints`. Stored on the model; passed to `watchProject` from `Init` and reload commands. No live re-detection if a project changes layout mid-session — restart required.

## Out of scope

- File-mtime fallback for the ledger "Updated" column when the roadmap has no timestamps. Skipped for simplicity; column is blank for iter-* sprints.
- Cross-checking `iter-workflow-status.txt` against parsed roadmap status. The overlay header shows both side by side; the user adjudicates.
- Rendering sub-iterations mentioned only in a parent's status string as separate list rows. They are visible inside the parent's section text.
- Inlining review file contents. We list filename + mtime only.
- Re-detecting layout mid-session when a `ledger.yaml` appears in a previously iter-* project.

## Testing strategy

TDD. Each new behavior gets a failing test first.

### `roadmap_test.go` (new)

Fixtures in `testdata/roadmap/`:
- `kanmail-roadmap.md` — trimmed slice of real kanmail roadmap with `done`, `pending`, `in_progress (...)`, `done:ITER-XXX` variants.
- `door-roadmap.md` — trimmed slice of real door roadmap with `ITER-0003a/b` sub-iter sections.
- `roadmap-no-status.md` — 4 ITER headings, no Status lines. Exercises inference path.
- `roadmap-malformed.md` — headings with various dashes, weird whitespace, missing titles.

Test cases:
- `TestParseRoadmap_KanmailFixture` — count, IDs, titles, section bodies preserved.
- `TestParseRoadmap_StatusNormalization` — table-driven mapping check.
- `TestParseRoadmap_NoStatusLines` — every Sprint has `Status = ""`.
- `TestParseRoadmap_Subiterations` — `ITER-0006a-i`, `ITER-0009b`, `ITER-0003a` parse cleanly.
- `TestParseRoadmap_DashVariants` — accept `—`, `–`, `-`.
- `TestParseRoadmap_Empty` — empty input → no error, empty slice.

### `ledger_test.go` (extended)

Fixture `testdata/iter-project/`:
- `docs/iterations/roadmap.md` (small synthetic)
- `.ai/iter-current-iteration.txt` → `ITER-0003a`
- `.ai/iter-workflow-status.txt` → `fail`
- `.ai/iter-loop-context.txt` → a few narrative lines
- `.ai/iter-run-temp/task_status.txt` → `TASK-01|done\nTASK-02|in_progress`
- `.ai/iter-run-temp/scope_review_a.md` (empty, just for listing)
- `docs/iterations/progress.md` mentioning `ITER-0003a`

Tests:
- `TestLoadSprints_IterLayout` — list returned, current iter flagged, inferred statuses correct.
- `TestLoadSprints_IterLayoutFallbackPath` — same fixture under `.ai/iter-scope-temp/` instead of `docs/iterations/`.
- `TestLoadSprints_PrefersLedgerOverRoadmap` — both present; ledger wins.
- `TestLoadDetailContent_IterCurrent` — overlay blocks in status-first order, header carries both parsed + run status.
- `TestLoadDetailContent_IterNonCurrent` — section text only.
- `TestLoadDetailContent_OverlayMissingFiles` — delete overlay files; render still succeeds with blocks omitted.
- `TestLoadDetailContent_ProgressIterMismatch` — `progress.md` references wrong ITER; block skipped.

### `watcher_test.go` (extended)

- `TestWatchProject_IterLayout_TriggersOnRoadmapChange` — touch roadmap.md, receive `ledgerChangedMsg`.
- `TestWatchProject_IterLayout_TriggersOnCurrentIterChange` — touch iter-current-iteration.txt.
- `TestWatchProject_IterLayout_IgnoresExtractTemp` — touch `.ai/iter-extract-temp/chunk_001.md`, no message within 200ms.

### Real-project smoke

Add a row to `scenario_real_projects_test.go` (gitignored) pointing at the actual kanmail and door `.ai/` paths. `LoadSprints` returns at least N sprints and no error.

### Migration of existing tests

`LoadDetailContent(s)` → `LoadDetailContent(s, projectRoot)`: mechanical fix in `model.go` and any direct test call sites. All existing tests must keep passing.

## Implementation phases

The implementation plan (produced separately by the `writing-plans` skill) will sequence phases roughly as:

1. Parser + `Sprint` struct fields + tests (no integration yet)
2. `LoadSprints` third branch + inference pass + tests
3. `LoadDetailContent` signature change + non-current iter render + tests
4. Overlay composer + tests
5. Watcher layout-aware refactor + tests
6. Wire-up in `model.go` and `main.go`; manual smoke against real projects
