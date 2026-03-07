# Implementation Plan: Grouped Review Mode with Viewed/Commented Indicators

Add a `--groups` flag to organize files into named feature groups in the tree sidebar, with per-file "viewed" tracking, "commented" indicators, and a "Mark as viewed" button that advances to the next unviewed file. All indicators work in both grouped and ungrouped modes.

## Context

**Research Document**: `docs/research/2026-03-06-grouped-review-mode.md`

**Key Files**:
- `main.go` - CLI flag definitions, Config construction
- `internal/app/model.go` - All data types (TreeItem, ReviewModel, Config, Comment, File)
- `internal/app/tree.go` - `buildTree()` function, tree helpers
- `internal/app/app.go` - `Run()`, `buildLiveHandler()`, all event handlers
- `internal/app/view.go` - `updateView()`, view building functions
- `internal/app/io_helpers.go` - File loading, diff reading, TOON output
- `internal/ui/template.html` - Single-page live view template
- `internal/ui/styles.css` - All CSS styles
- `internal/app/http_render_test.go` - Render tests (7 buildTree call sites)

**Architectural Notes**:
- Uses `jfyne/live` for server-rendered live views over WebSocket — full re-render on every event
- Tree is a flat `[]TreeItem` list with `Depth` for indentation, rebuilt on file selection
- Two modes: `ModeFile` and `ModeDiff` — grouping is orthogonal and works with both
- Event handlers follow pattern: `getModel()` → mutate → `updateView()` → return model

**Functional Requirements** (EARS notation):
- When `--groups` flag is provided with a JSON file path, the system shall parse the file as an ordered array of `{name, files}` objects and organize the tree sidebar by groups
- When a file is not assigned to any group, the system shall place it in an auto-created "Other" group at the bottom
- When a user clicks "Mark as viewed" on a file, the system shall toggle the viewed state and navigate to the next unviewed file in the current group, then the next group
- When all files are marked as viewed, the system shall stay on the current file
- While a file within a group is selected, the system shall highlight the group header in the tree
- The system shall show a checkmark indicator next to viewed files in the tree
- The system shall show a comment dot indicator next to files with comments in the tree
- When a comment is added or deleted, the system shall update the comment indicator in the tree

## Batch Size

| Metric | Count | Rating |
|--------|-------|--------|
| Tasks | 11 | Large |
| Files | 12 | Large |
| Stages | 3 | Large |

**Overall: Large** (proceeding as single plan per user decision)

## Execution Stages

### Stage 1: Data Model, Types, and Tree Building

#### Test Creation Phase (parallel)
- T-test-1A: Write tests for `ParseGroupsFile` and group validation (hmm-test-writer)
  - Regression tests: existing `ParseRangeFlag` tests pass unchanged
  - New feature tests (RED): valid JSON parsing, invalid JSON error, empty name error, empty files error, file-not-found error
- T-test-1B: Write tests for `buildTree` with viewed/commented fields and `buildGroupedTree` (hmm-test-writer)
  - Regression tests: existing tree tests compile with new signature (nil viewed, nil comments)
  - New feature tests (RED): viewed indicator set, hasComments indicator set, group headers produced, "Other" group for ungrouped files, group ordering preserved
- T-test-1C: Write tests for `fileHasComments` helper (hmm-test-writer)
  - New feature tests (RED): returns true when comment exists for path, false otherwise, nil comments safe

#### Implementation Phase (sequential, depends on Test Creation Phase)
- T-impl-1A: Add Group type, Config.Groups, ReviewModel fields, ParseGroupsFile (hmm-implement-worker, TDD mode)
  - Files: `internal/app/model.go`, `internal/app/io_helpers.go`
  - Make RED tests pass (GREEN)
- T-impl-1B: Add TreeItem fields, fileHasComments, update buildTree signature, implement buildGroupedTree, update test call sites (hmm-implement-worker, TDD mode, depends on T-impl-1A)
  - Files: `internal/app/tree.go`, `internal/app/http_render_test.go`
  - Make RED tests pass (GREEN)
  - Includes updating all 7 buildTree call sites in http_render_test.go

### Stage 2: App Layer — CLI Flag, Run(), rebuildTree, Comment Handler Updates

#### Test Creation Phase (parallel)
- T-test-2A: Write tests for `nextUnviewedFile` navigation logic (hmm-test-writer)
  - New feature tests (RED): ungrouped next file, ungrouped wrap-around, ungrouped all viewed, grouped cross-group advance, grouped within-group advance, nil viewed map, selected path not found

#### Implementation Phase (sequential, depends on Test Creation Phase)
- T-impl-2A: Add --groups flag to main.go, add rebuildTree helper, update all buildTree call sites in app.go, add rebuildTree to comment handlers, initialize new ReviewModel fields in Run() (hmm-implement-worker, TDD mode)
  - Files: `main.go` (modifies), `internal/app/app.go` (modifies)
- T-impl-2B: Add mark-viewed event handler with nextUnviewedFile and selectFile helpers, refactor select-file handler to use selectFile (hmm-implement-worker, TDD mode, depends on T-impl-2A)
  - Files: `internal/app/app.go` (modifies)
  - Make RED tests pass (GREEN)

### Stage 3: UI Layer — Template and CSS (depends on Stage 2)

#### Test Creation Phase
(No separate test creation — UI changes are verified by existing render tests and manual meatcheck review)

#### Implementation Phase (parallel)
- T-impl-3A: Update template.html — group headers, viewed/commented indicators in tree, mark-as-viewed button (hmm-implement-worker)
- T-impl-3B: Update styles.css — group header CSS, indicator CSS, content footer CSS, .main grid update (hmm-implement-worker)
- T-impl-3C: Update help text, skill.md documentation (hmm-implement-worker)

## Task List

### Data Model & Types

- [ ] Add `Group` type and extend `Config` and `ReviewModel` (`internal/app/model.go`) [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - Add `Group` struct: `Name string \`json:"name"\``, `Files []string \`json:"files"\``
  - Add `Groups []Group` to `Config` struct
  - Add to `ReviewModel`: `Viewed map[string]bool`, `Groups []Group`, `HasGroups bool`
  - Add to `TreeItem`: `IsGroup bool`, `Viewed bool`, `HasComments bool`, `GroupName string`, `GroupActive bool`

- [ ] Add `ParseGroupsFile` function (`internal/app/io_helpers.go`) [Stage 1]
  - Files: `internal/app/io_helpers.go` (modifies)
  - `func ParseGroupsFile(path string) ([]Group, error)` reads JSON file, unmarshals into `[]Group`
  - Validate: non-empty Name, non-empty Files per group
  - Add `"encoding/json"` to imports
  - Normalize file paths with `filepath.ToSlash(filepath.Clean(f))` to match model paths

### Tree Building

- [ ] Add `fileHasComments` helper and update `buildTree` signature (`internal/app/tree.go`) [Stage 1]
  - Files: `internal/app/tree.go` (modifies)
  - Add `func fileHasComments(path string, comments []Comment) bool`
  - Change `buildTree` signature to `buildTree(files []File, selectedPath string, viewed map[string]bool, comments []Comment) []TreeItem`
  - In the walk closure, set `item.Viewed = viewed[n.File.Path]` and `item.HasComments = fileHasComments(n.File.Path, comments)` for file nodes
  - Handle nil `viewed` map (Go map read on nil returns zero value, so no nil check needed)

- [ ] Implement `buildGroupedTree` function (`internal/app/tree.go`) [Stage 1]
  - Files: `internal/app/tree.go` (modifies)
  - `func buildGroupedTree(groups []Group, files []File, selectedPath string, viewed map[string]bool, comments []Comment) []TreeItem`
  - Iterate groups in order; for each group emit a `TreeItem{Name: group.Name, IsGroup: true, Depth: 0, GroupActive: selectedPath belongs to this group}`
  - Compute `GroupActive` by checking if `selectedPath` matches any file path in the group
  - For each file in the group, find it in files slice, emit `TreeItem{Name: basename, Path: filePath, Depth: 1, Selected: filePath == selectedPath, Viewed: viewed[filePath], HasComments: fileHasComments(filePath, comments), GroupName: group.Name}`
  - Collect ungrouped files into "Other" group at bottom
  - Use `filepath.Base()` for display name

### App Layer

- [ ] Add `--groups` flag, group loading, rebuildTree helper, and update all call sites (`main.go`, `internal/app/app.go`) [Stage 2]
  - Files: `main.go` (modifies), `internal/app/app.go` (modifies)
  - In `main.go`: add `groups = flag.String("groups", "", "path to JSON file with ordered file groups")`, after `ParseRangeFlag` call `app.ParseGroupsFile(*groups)` if non-empty, pass `Groups: parsedGroups` into `cfg`
  - In `app.go`: create `func rebuildTree(model *ReviewModel)` that gets file list (diffFilesAsFiles or model.Files based on mode), calls `buildGroupedTree()` if `model.HasGroups` else `buildTree()`, passes `model.Viewed` and `model.Comments`
  - Replace 4 `model.Tree = buildTree(...)` calls (lines 94, 97, 208, 218) with `rebuildTree(model)`
  - Add `rebuildTree(model)` calls in `add-comment`, `delete-comment`, and `edit-comment` handlers (before `updateView`)
  - Initialize `Viewed: make(map[string]bool)`, `Groups: cfg.Groups`, `HasGroups: len(cfg.Groups) > 0` in ReviewModel construction in `Run()`

- [ ] Add `mark-viewed` event handler with navigation (`internal/app/app.go`) [Stage 2]
  - Files: `internal/app/app.go` (modifies)
  - Add `func nextUnviewedFile(model *ReviewModel) string`:
    - Build ordered file path list: if HasGroups, iterate Groups in order (each group's Files); if not grouped, use `model.Files` paths (file mode) or `model.DiffFiles` paths (diff mode)
    - Find current file's index, start from next, wrap around
    - Return first path where `model.Viewed[path]` is false, or `""` if all viewed
  - Add `func selectFile(model *ReviewModel, path string)` helper:
    - Sets SelectedPath, resets CodeViewKey/SelectionStart/SelectionEnd/Error
    - Calls `rebuildTree(model)` and `updateView(model)`
  - Register `"mark-viewed"` event handler:
    - Toggle `model.Viewed[model.SelectedPath]`
    - If just marked viewed: call `nextUnviewedFile`, navigate via `selectFile` if found
    - If just unmarked: `rebuildTree(model)` + `updateView(model)`
  - Update existing `select-file` handler to use `selectFile` helper

### UI Layer

- [ ] Update tree rendering in template (`internal/ui/template.html`) [Stage 3]
  - Files: `internal/ui/template.html` (modifies)
  - Replace two-branch tree rendering with three-branch: `{{if .IsGroup}}...{{else if .IsDir}}...{{else}}...{{end}}`
  - Group header: `<div class="tree-item group{{if .GroupActive}} active-group{{end}}">{{.Name}}</div>`
  - File items: wrap name in `<span class="tree-name">`, add `<span class="tree-indicators">` with comment dot (&#9679;) and viewed checkmark (&#10003;)
  - Add viewed/has-comments CSS classes to file tree item div
  - Add content footer below `</main>` with "Mark as viewed" button: `live-click="mark-viewed"`
  - Button text toggles: "Mark as viewed" / "Viewed &#10003;"

- [ ] Add CSS for groups, indicators, and content footer (`internal/ui/styles.css`) [Stage 3]
  - Files: `internal/ui/styles.css` (modifies)
  - `.tree-item.group` — bold, uppercase, border-bottom separator
  - `.tree-item.group.active-group` — accent color highlight
  - `.tree-indicators` — inline-flex, right-aligned, gap
  - `.viewed-check` — green color (#2ea043)
  - `.comment-dot` — accent color, smaller font
  - `.tree-item.viewed .tree-name` — dimmed (var(--muted))
  - `.content-footer` — flex, border-top, panel background
  - Update `.main` grid-template-rows to `auto 1fr auto`
  - Add `justify-content: space-between` to `.tree-item.file`

- [ ] Update help text and skill documentation [Stage 3]
  - Files: `internal/app/app.go` (modifies), `internal/app/skill.md` (modifies)
  - Add `--groups` to `PrintHelp()` flags list and usage examples
  - Add `--groups` usage and JSON format to skill.md

### Tests

- [ ] Write tests for ParseGroupsFile (`internal/app/groups_test.go`) [Stage 1]
  - Files: `internal/app/groups_test.go` (creates)
  - Valid JSON array with multiple groups
  - Invalid JSON returns error
  - Group with empty name returns error
  - Group with empty files list returns error
  - Non-existent file path returns error

- [ ] Write tests for buildTree viewed/commented and buildGroupedTree (`internal/app/tree_test.go`) [Stage 1]
  - Files: `internal/app/tree_test.go` (creates)
  - buildTree: viewed=true sets Viewed on item, comment exists sets HasComments, nil viewed safe
  - buildGroupedTree: group headers produced with IsGroup/Depth=0, files at Depth=1 with GroupName, "Other" group for ungrouped, ordering preserved, selected file correct

- [ ] Write tests for nextUnviewedFile navigation (`internal/app/viewed_test.go`) [Stage 2]
  - Files: `internal/app/viewed_test.go` (creates)
  - Ungrouped: next unviewed, wrap-around, all viewed returns ""
  - Grouped: within-group advance, cross-group advance, all viewed
  - Edge cases: nil viewed map, selected path not found

## Acceptance Criteria

~~~gherkin
Feature: Grouped review mode with viewed and commented indicators

  Scenario: Groups flag loads and organizes files into groups
    Given a groups JSON file with groups "Auth" containing "auth.go" and "API" containing "handler.go"
    And meatcheck is invoked with --groups groups.json and files auth.go handler.go utils.go
    When the UI loads
    Then the tree sidebar shows group headers "Auth" and "API" in order
    And "auth.go" appears under "Auth"
    And "handler.go" appears under "API"
    And "utils.go" appears under an "Other" group at the bottom

  Scenario: Groups work with diff mode
    Given a groups JSON file with group "Frontend" containing "src/app.tsx" and group "Backend" containing "api/handler.go"
    And a unified diff file containing changes to "src/app.tsx", "api/handler.go", and "config.yaml"
    And meatcheck is invoked with --groups groups.json --diff changes.diff
    When the UI loads
    Then the tree sidebar shows group headers "Frontend" and "Backend" in order
    And "src/app.tsx" appears under "Frontend"
    And "api/handler.go" appears under "Backend"
    And "config.yaml" appears under an "Other" group at the bottom

  Scenario: Marking a file as viewed shows indicator and advances
    Given a file is selected in the review UI
    When the user clicks "Mark as viewed"
    Then a green checkmark appears next to the file in the tree
    And the file name is dimmed in the tree
    And the UI navigates to the next unviewed file

  Scenario: Mark as viewed advances across groups
    Given grouped mode with groups A and B
    And the last unviewed file in group A is selected
    When the user clicks "Mark as viewed"
    Then the UI navigates to the first unviewed file in group B

  Scenario: All files viewed stays on current file
    Given all files except the current one are marked as viewed
    When the user clicks "Mark as viewed" on the last unviewed file
    Then the file is marked as viewed
    And the UI stays on the current file

  Scenario: Comment indicator appears in tree after adding comment
    Given a file with no comments
    When the user adds a comment on a line in the file
    Then a comment dot indicator appears next to the file in the tree

  Scenario: Comment indicator disappears when last comment is deleted
    Given a file with exactly one comment
    When the user deletes the comment
    Then the comment dot indicator is removed from the file in the tree

  Scenario: Active group is highlighted when file is selected
    Given grouped mode with multiple groups
    When the user selects a file within a group
    Then the group header for that file's group is highlighted

  Scenario: Viewed and commented indicators work in ungrouped mode
    Given meatcheck is invoked without --groups flag
    When the user marks a file as viewed and adds a comment to another file
    Then the viewed checkmark appears on the viewed file
    And the comment dot appears on the commented file

  Scenario: Unmarking a viewed file removes indicator
    Given a file that has been marked as viewed
    When the user clicks the "Viewed" button on that file
    Then the checkmark is removed from the file in the tree
    And the file name is no longer dimmed
    And the button text changes back to "Mark as viewed"
    And the UI stays on the current file

  Scenario: Invalid groups JSON produces error
    Given a groups JSON file with invalid syntax
    When meatcheck is invoked with --groups pointing to it
    Then the CLI exits with an error message describing the JSON problem
~~~

**Source**: Generated from plan context and research decisions

## Implementation Notes

- **Path normalization**: Group file paths in JSON should be normalized with `filepath.ToSlash(filepath.Clean(f))` during parsing to match the path format used in `File.Path` and `DiffFile.Path`.
- **Nil-safe map access**: Go map reads on nil map return zero value (false for bool), so `viewed[path]` is safe even when `Viewed` is nil. However, `Viewed` must be initialized to non-nil in `Run()` because the `mark-viewed` handler writes to it.
- **Template nil map**: `{{index .Viewed .SelectedPath}}` panics on nil map. The `Viewed` map must be initialized before first render.
- **Performance**: `fileHasComments` is O(n) per file. Fine for meatcheck scale (tens of files/comments).
- **Tree rebuild in comment handlers**: Adding `rebuildTree(model)` to add-comment/delete-comment/edit-comment handlers ensures HasComments indicators stay current. Edit-comment doesn't change which files have comments but calling rebuildTree is harmless and consistent.

## Refs

- `docs/research/2026-03-06-grouped-review-mode.md` — Research document with architecture analysis and design decisions
