# Implementation Plan: Diff View Options (Unified/Side-by-Side, Preferences, Background)

Add switchable unified/side-by-side diff view with localStorage persistence, enable commenting on old (deleted) lines, and change the diff area background to black.

## Context

**Research Document**: `docs/research/2026-03-09-diff-view-options.md`

**Key Files**:
- `internal/app/model.go` - Data types: `ReviewModel`, `ViewDiffLine`, `ViewDiffHunk`, `ViewDiffFile`, `Comment`
- `internal/app/view.go` - View builders: `buildViewDiff()`, `updateDiffView()`, `projectLineComments()`, `diffLineExists()`
- `internal/app/app.go` - Event handlers: `toggle-file-render`, `select-line`, `add-comment`, `cancel-comment`
- `internal/app/diff.go` - Diff parser: `parseUnifiedDiff()`, `DiffLine`, `DiffHunk`
- `internal/ui/template.html` - HTML template with JS hooks for line selection and localStorage
- `internal/ui/styles.css` - CSS with theme variables and diff styling

**Architectural Notes**:
- Server-driven reactive UI via `jfyne/live` — events go client→server, server mutates model, full re-render diffs to client
- All UI state lives in `ReviewModel` struct; only sidebar width persists in localStorage
- Template uses Go `html/template` with `{{if}}` conditionals for mode branching
- CSS uses custom properties (`:root` variables), embedded in binary via `//go:embed`

**Functional Requirements** (EARS notation):
- When the user clicks the diff format toggle, the system shall switch between unified and side-by-side diff views
- While in side-by-side mode, the system shall display old lines on the left and new lines on the right, paired by position within each hunk
- When the user selects a diff format, the system shall persist the choice to localStorage
- When the application loads, the system shall restore the diff format from localStorage
- While viewing a diff in either format, the system shall allow clicking on both old (deleted) and new (added) lines to create comments
- The diff code area background shall be black (#000000)
- If the user is not in diff mode, then the diff format toggle shall be hidden

## Execution Stages

### Stage 1: Data Model and Background Color

#### Test Creation Phase (parallel)
- T-test-1A: Write tests for `DiffFormat` model types and `diffOldLineExists` (hmm-test-writer)
  - New feature tests (RED): Scenarios 1, 2
- T-test-1B: Write tests for side-aware `projectLineComments` (hmm-test-writer)
  - Regression tests: existing `projectLineComments` behavior with empty side
  - New feature tests (RED): Scenarios 5, 6

#### Implementation Phase (parallel, depends on Test Creation Phase)
- T-impl-1A: Add `DiffFormat` type, `Comment.Side`, `SelectionSide`, `ViewDiffRow` types to model.go (hmm-implement-worker, TDD mode)
  - Files: `internal/app/model.go` (modifies)
  - Make RED tests pass (GREEN)
- T-impl-1B: Update `projectLineComments` for side-aware matching, add `diffOldLineExists` (hmm-implement-worker, TDD mode)
  - Files: `internal/app/view.go` (modifies)
  - Make RED tests pass (GREEN)
- T-impl-1C: Change `.diff` background to black (hmm-implement-worker, TDD mode)
  - Files: `internal/ui/styles.css` (modifies)

### Stage 2: Event Handlers and Unified View Old-Line Support (depends on Stage 1)

#### Test Creation Phase (parallel)
- T-test-2A: Write tests for `buildViewDiff` with old-line selection and commenting (hmm-test-writer)
  - Regression tests: existing `buildViewDiff` behavior
  - New feature tests (RED): Scenarios 3, 4

#### Implementation Phase (sequential, depends on Test Creation Phase)
- T-impl-2A: Update all app.go event handlers — add `toggle-diff-format`, `init-diff-format`; update `select-line`, `add-comment`, `cancel-comment` for old-line support; initialize `DiffFormat` in `Run()` (hmm-implement-worker, TDD mode)
  - Files: `internal/app/app.go` (modifies)
- T-impl-2B: Update `buildViewDiff` and `updateDiffView` for old-line selection/commenting and DiffFormat dispatch (hmm-implement-worker, TDD mode)
  - Files: `internal/app/view.go` (modifies)
  - Make RED tests pass (GREEN)

### Stage 3: Side-by-Side View Builder (depends on Stage 2)

#### Test Creation Phase (parallel)
- T-test-3A: Write tests for `buildViewDiffSplit` pairing algorithm (hmm-test-writer)
  - New feature tests (RED): Scenarios 7, 8, 9

#### Implementation Phase (parallel, depends on Test Creation Phase)
- T-impl-3A: Implement `buildViewDiffSplit` function (hmm-implement-worker, TDD mode)
  - Files: `internal/app/view.go` (modifies)
  - Make RED tests pass (GREEN)

### Stage 4: Template, CSS, and JavaScript (depends on Stage 3)

#### Test Creation Phase
- T-test-4A: Write render tests for toggle visibility and split template output (hmm-test-writer)
  - New feature tests (RED): Scenario 10

#### Implementation Phase (sequential, depends on Test Creation Phase)
- T-impl-4A: Update template.html — add toggle button, `data-old-line` attributes, side-by-side template block, old-line comment form conditional, JS click handler for old-line clicks, localStorage persistence (hmm-implement-worker, TDD mode)
  - Files: `internal/ui/template.html` (modifies)
- T-impl-4B: Add side-by-side CSS layout (hmm-implement-worker, TDD mode)
  - Files: `internal/ui/styles.css` (modifies)

## Task List

### Data Model

- [x] Add `DiffFormat` type and constants [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - Add `type DiffFormat string` with `DiffFormatUnified DiffFormat = "unified"` and `DiffFormatSplit DiffFormat = "split"`.
  - Add `DiffFormat DiffFormat` field to `ReviewModel` struct.

- [x] Add `Side` field to `Comment` struct [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - Add `Side string \`json:"side,omitempty"\`` after `EndLine`. Values: `""` (new, default), `"old"`.
  - `omitempty` ensures backward compatibility in TOON output.

- [x] Add `SelectionSide` field to `ReviewModel` [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - Add `SelectionSide string` near `SelectionStart`/`SelectionEnd`.

- [x] Add `ViewDiffRow` and `ViewDiffSide` types for side-by-side view [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - `ViewDiffSide`: `Line int`, `Kind DiffLineKind`, `Text string`, `HTML template.HTML`, `Empty bool`, `Selected bool`, `Commented bool`, `Comments []ViewComment`
  - `ViewDiffRow`: `Left ViewDiffSide`, `Right ViewDiffSide`
  - `ViewDiffSplitHunk`: `Header string`, `Rows []ViewDiffRow`
  - Add `ViewDiffSplit []ViewDiffSplitHunk` field to `ReviewModel`.

### View Building

- [x] Update `projectLineComments` for side-aware matching [Stage 1]
  - Files: `internal/app/view.go` (modifies)
  - Add `side string` parameter. Filter `c.Side` against the provided side. Treat `c.Side == ""` as `"new"` for backward compat.
  - Update all call sites: `buildViewDiff` (2 calls), `buildSingleViewLine` (1 call). File-mode calls pass `""`.

- [x] Add `diffOldLineExists` function [Stage 1]
  - Files: `internal/app/view.go` (modifies)
  - Same structure as `diffLineExists` but checks `dl.OldLine == oldLine && dl.Kind != DiffAdd`.

- [x] Update `buildViewDiff` for old-line selection and commenting [Stage 2]
  - Files: `internal/app/view.go` (modifies)
  - Accept `selectionSide string` parameter.
  - Remove the `selectable` guard that blocks deleted lines. All lines with OldLine > 0 or NewLine > 0 are selectable.
  - When `selectionSide == "old"`, check selection against `dl.OldLine`; otherwise `dl.NewLine`.
  - Project comments: call `projectLineComments` with side `""` on `dl.NewLine` and side `"old"` on `dl.OldLine`, merge results.

- [x] Update `updateDiffView` to dispatch on DiffFormat [Stage 2]
  - Files: `internal/app/view.go` (modifies)
  - When `DiffFormatSplit`, call `buildViewDiffSplit` and populate `model.ViewDiffSplit`.
  - When `DiffFormatUnified` (or default), use existing `buildViewDiff`.
  - Clear both `model.ViewDiff` and `model.ViewDiffSplit` at the start.
  - Pass `model.SelectionSide` to `buildViewDiff` (and later `buildViewDiffSplit`) for correct selection highlighting.

- [x] Implement `buildViewDiffSplit` pairing algorithm [Stage 3]
  - Files: `internal/app/view.go` (modifies)
  - For each hunk, pair lines into rows:
    - Context lines: both Left and Right populated with same text
    - Del/Add blocks: collect consecutive dels then adds, zip 1:1. Unpaired lines get `Empty: true` opposite side.
  - Apply selection/comment projection to each side independently.
  - Render syntax highlighting for old and new lines separately.

### Event Handlers

- [x] Update all app.go handlers and initialization for diff format and old-line support [Stage 2]
  - Files: `internal/app/app.go` (modifies)
  - **New handlers**:
    - `toggle-diff-format`: toggle `model.DiffFormat` between `DiffFormatUnified` and `DiffFormatSplit`. Clear selection state (`SelectionStart`, `SelectionEnd`, `SelectionSide`). Call `updateView(model)`.
    - `init-diff-format`: read `p.String("format")`, validate (`"unified"` or `"split"`), set `model.DiffFormat`. Call `updateView(model)`.
  - **Modified handlers**:
    - `select-line`: read `old_line` param. When `old_line > 0`, validate via `diffOldLineExists`, set `model.SelectionSide = "old"`. When selecting new lines, set `model.SelectionSide = ""`.
    - `add-comment`: set `Side: model.SelectionSide` on new `Comment`. Clear `model.SelectionSide` after adding.
    - `cancel-comment`: add `model.SelectionSide = ""` alongside existing selection clearing.
  - **Initialization**: set `DiffFormat: DiffFormatUnified` in `ReviewModel` struct literal in `Run()`.

### Template and UI

- [x] Update template.html for diff format toggle, side-by-side view, old-line support, and localStorage [Stage 4]
  - Files: `internal/ui/template.html` (modifies)
  - **Toggle button**: Add diff format toggle in `.column-header`, only visible when `$root.Mode == "diff"`. Uses `live-click="toggle-diff-format"`. SVG icon of two rectangles side by side. `active` class when `$root.DiffFormat == "split"`.
  - **Unified view old-line support**: Add `data-old-line="{{.OldLine}}"` to `.diff-line` div (line 123). Show comment threads on deleted lines — update template conditional so comments render when OldLine > 0 and the comment has side `"old"`. Update the inline-comment form conditional (template line 136) to also trigger when `$root.SelectionSide == "old"` and `.OldLine == $root.SelectionEnd`.
  - **Side-by-side template block**: Add `{{if eq $root.DiffFormat "split"}}` conditional inside the diff mode block. Each row renders as `.diff-row-split` with two `.diff-cell` halves (left=old with `data-old-line`, right=new with `data-line`). Empty sides render placeholder cells. Comment threads render full-width below rows. Inline comment form spans full width after selection end row.
  - **JS click handler**: If `data-line` (NewLine) is 0, fall back to `data-old-line` (OldLine). Send `old_line: "1"` flag to server when clicking an old line.
  - **localStorage persistence**: On mount, read `localStorage.getItem("meatcheck-diff-format")` and send `init-diff-format` event (deferred until `window.Live` is available via polling). On toggle, save via `handleEvent("diff-format-changed", ...)`.

### CSS

- [x] Change `.diff` background to black [Stage 1]
  - Files: `internal/ui/styles.css` (modifies)
  - Change line 777 from `background: var(--panel)` to `background: #000000`.

- [x] Add side-by-side CSS layout [Stage 4]
  - Files: `internal/ui/styles.css` (modifies)
  - `.diff-row-split`: `display: grid; grid-template-columns: 1fr 1fr`
  - `.diff-cell`: `display: grid; grid-template-columns: 4ch 1ch max-content; gap: 12px`
  - Kind-based backgrounds on `.diff-cell[data-kind="add"]` and `[data-kind="del"]`
  - Hover, selected, commented states on `.diff-cell`
  - `.diff-cell .code-text` rules mirroring existing `.diff-line .code-text`
  - Border separator between left and right cells

## Acceptance Criteria

~~~gherkin
Feature: Diff view options

  Scenario: Toggle between unified and side-by-side diff view
    Given the user is viewing a diff
    When the user clicks the diff format toggle button
    Then the view switches from unified to side-by-side (or vice versa)
    And the toggle button shows an active state when in side-by-side mode

  Scenario: Diff format preference persists across sessions
    Given the user has selected side-by-side diff format
    When the user closes and reopens the tool
    Then the diff view loads in side-by-side format

  Scenario: Side-by-side view pairs old and new lines correctly
    Given a diff hunk with 2 deleted lines followed by 3 added lines
    When the view is in side-by-side mode
    Then the first 2 rows show del on the left and add on the right
    And the 3rd row shows an empty left cell and add on the right

  Scenario: Context lines appear on both sides
    Given a diff hunk with context lines
    When the view is in side-by-side mode
    Then context lines appear on both the left and right sides with matching line numbers

  Scenario: Comment on a deleted line in unified view
    Given the user is viewing a diff in unified mode
    When the user clicks on a deleted line
    Then the line is selected and a comment form appears
    And the comment is anchored to the old-side line number

  Scenario: Comment on a deleted line in side-by-side view
    Given the user is viewing a diff in side-by-side mode
    When the user clicks on a deleted line in the left column
    Then the line is selected and a comment form appears
    And the comment is anchored to the old-side line number

  Scenario: Side-by-side comments on old vs new lines with same number
    Given a comment exists on old line 5 (deleted) and another on new line 5 (added)
    When viewing in side-by-side mode
    Then the old-line comment appears in the left column
    And the new-line comment appears in the right column
    And they do not cross-contaminate

  Scenario: Side-by-side view renders hunk headers full-width
    Given a diff with multiple hunks
    When the view is in side-by-side mode
    Then hunk headers span the full width of both columns

  Scenario: Side-by-side view with syntax highlighting
    Given syntax highlighting is enabled
    When the view is in side-by-side mode
    Then both left and right columns render syntax-highlighted code

  Scenario: Diff format toggle hidden in file mode
    Given the user is viewing a plain file (not a diff)
    Then the diff format toggle button is not visible

  Scenario: Diff area background is black
    Given the user is viewing a diff
    Then the diff code area background is black (#000000)
    And the sidebar and other panels retain their original background
~~~

**Source**: Generated from plan context

## Implementation Notes

- **Backward compatibility**: `Comment.Side` uses `omitempty` so existing TOON output is unchanged. Empty `Side` is treated as `"new"`.
- **Shift-click range selection**: Old-line selection does not support shift-click ranges initially. Shift-clicking resets to single-line selection for simplicity.
- **Syntax highlighting in split view**: Old-side and new-side lines are rendered separately through `codeRenderer.RenderLines` for accurate highlighting.
- **Performance**: Side-by-side doubles DOM elements per row. Acceptable for typical diff sizes.
- **localStorage init timing**: Uses polling interval (50ms) to wait for `window.Live` to be available before sending `init-diff-format`.

## Refs

- `docs/research/2026-03-09-diff-view-options.md`
