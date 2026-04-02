# Implementation Plan: Per-List-Item Commenting in Markdown Review

Enable users to click on individual markdown list items and leave comments anchored to that item's line range. Currently, the entire list is treated as a single commentable block.

## Context

**Research Document**: `docs/research/2026-04-02-list-item-commenting.md`

**Architectural Notes**:
- Server-side LiveView (`github.com/jfyne/live`) — all state lives on `ReviewModel`, events are handled server-side, and the full template re-renders on each event.
- Goldmark (with GFM extension) parses markdown into an AST. `renderMarkdownBlocks()` walks top-level AST children, producing one `MarkdownBlock` per node. An `ast.List` node containing multiple `ast.ListItem` children is treated as a single block.
- Rendering an `ast.ListItem` in isolation produces `<li>...</li>` without a `<ul>`/`<ol>` wrapper. Tight/loose behavior and GFM task checkboxes work correctly in isolation.
- `nodeByteRange()` correctly isolates per-item byte ranges. `byteToLine()` maps them to 1-based line numbers.
- `projectBlockComments()` uses line-range overlap, so legacy comments spanning full list ranges will still display correctly when the list is split.
- `Element.closest("[data-line]")` in the JS click handler finds the innermost match, so per-item `data-line` attributes take precedence over any parent — no JS changes needed.

**Functional Requirements** (EARS notation):
- When a user clicks on a list item in rendered markdown view, the system shall select only that item's line range (not the entire list).
- When a user submits a comment while a single list item is selected, the system shall anchor the comment to that item's StartLine and EndLine.
- When a rendered markdown file contains a list, the system shall display each list item as an independently clickable and commentable block.
- When a list item has a comment, the system shall display the comment thread directly below that item (not after the entire list).
- While rendering per-item blocks, the system shall wrap consecutive list-item blocks in the correct `<ul>` or `<ol>` HTML tag for proper list styling.
- If an ordered list starts at a number other than 1, the system shall include the `start` attribute on the `<ol>` tag.
- When rendering GFM task list items as individual blocks, the system shall correctly display checkboxes.

**Key Files**:
- `internal/app/model.go` — `MarkdownBlock` struct definition (lines 50-57)
- `internal/app/assets.go` — `renderMarkdownBlocks()` (lines 115-195), `nodeByteRange()` (lines 199-240)
- `internal/app/view.go` — `projectBlockComments()` (lines 339-358), `updateFileView()` (lines 20-59)
- `internal/ui/template.html` — Markdown block rendering (lines 222-247), JS click handler (lines 338-359)
- `internal/ui/styles.css` — `.md-block` styling (lines 329-360), `.markdown ul/ol` (lines 576-582)
- `internal/app/markdown_test.go` — Existing tests for markdown block rendering

## Acceptance Criteria

~~~gherkin
Feature: Per-list-item commenting in markdown review

  Scenario: Click selects individual list item
    Given a markdown file with a 3-item unordered list on lines 5-7
    When the user clicks on the second list item
    Then only line 6 is selected
    And the first and third items are not selected

  Scenario: Comment anchors to single list item
    Given a markdown file with a 3-item list
    And the user has selected the second item (line 6)
    When the user submits a comment "Fix this item"
    Then the comment is created with StartLine=6 and EndLine=6
    And the comment thread appears below the second item only

  Scenario: Ordered list preserves numbering
    Given a markdown file with an ordered list starting at 3
    When the file is rendered in markdown preview mode
    Then the list items are numbered starting from 3
    And each item is independently clickable

  Scenario: GFM task list checkboxes render correctly
    Given a markdown file containing "- [ ] todo" and "- [x] done"
    When the file is rendered in markdown preview mode
    Then each task item shows its checkbox (unchecked or checked)
    And each task item is independently commentable

  Scenario: Nested lists stay within parent item
    Given a markdown file with a list item containing a nested sub-list
    When the file is rendered in markdown preview mode
    Then the parent item's block includes the nested list HTML
    And the nested items are not split into separate commentable blocks

  Scenario: Non-list blocks are unaffected
    Given a markdown file with headings, paragraphs, and blockquotes
    When the file is rendered in markdown preview mode
    Then these blocks render exactly as before
    And they carry no list wrapper metadata

  Scenario: Shift-click selects range across list items
    Given a markdown file with a 3-item list
    And the user has clicked item 1 (line 5)
    When the user shift-clicks item 3 (line 7)
    Then lines 5-7 are selected (spanning all three items)

  Scenario: Legacy comment on full list range displays correctly
    Given a comment anchored to lines 5-7 (the full list range)
    And the list is now rendered as 3 per-item blocks
    Then all three items show the "commented" visual indicator
    And the comment text displays below the first item
~~~

**Source**: Generated from plan context

## Batch Size

| Metric | Count | Rating |
|--------|-------|--------|
| Tasks | 5 | Small |
| Files | 5 | Small |
| Stages | 1 | Small |

**Overall: Small-Medium**

## Task List

### Backend: Model and Rendering

- [ ] **T1**: Add `ListOpen` and `ListClose` fields to `MarkdownBlock` struct (`internal/app/model.go`) [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - Add `ListOpen template.HTML` and `ListClose template.HTML` to the `MarkdownBlock` struct. `ListOpen` holds the opening list tag (e.g., `<ul>`, `<ol>`, `<ol start="3">`). `ListClose` holds the closing tag (`</ul>` or `</ol>`). For non-list blocks, both are zero-value empty strings. For the first item in a list, `ListOpen` is set. For the last item, `ListClose` is set. A single-item list has both on the same block.
  - Must be `template.HTML` (not `string`) so Go's `html/template` does not escape the angle brackets.

- [ ] **T2**: Modify `renderMarkdownBlocks()` to split `ast.List` into per-item blocks (`internal/app/assets.go`) [Stage 1, depends: T1]
  - Files: `internal/app/assets.go` (modifies)
  - In the main loop (line 163), add a type-switch on `child`. When `child` is `*ast.List`:
    1. Build the opening tag: `<ul>` for unordered, `<ol>` for ordered (with `start="N"` if `list.IsOrdered() && list.Start != 1`; omit `start` attribute when `Start == 1` as it is the default).
    2. Iterate `listNode.FirstChild()` → `NextSibling()` to walk `ast.ListItem` children.
    3. For each item: compute `startLine`/`endLine` via `nodeByteRange()` + `byteToLine()`, render via `r.Render(&buf, source, item)`, apply `rewriteMarkdownImageSources()`.
    4. Set `ListOpen` on the first item's block, `ListClose` on the last item's block.
    5. Update `lastLine` after each item block.
  - For non-list nodes, continue with existing block creation (with empty `ListOpen`/`ListClose`).
  - Do NOT use `ast.AppendChild()` to build temp wrapper nodes — it mutates the original AST.

### Frontend: Template and CSS

- [ ] **T3**: Emit list wrapper tags in the template (`internal/ui/template.html`) [Stage 1, depends: T1]
  - Files: `internal/ui/template.html` (modifies)
  - In the `{{range .ViewFile.MarkdownBlocks}}` loop (lines 224-246):
    - Before `<div class="md-block ...">`: emit `{{if .ListOpen}}{{.ListOpen}}{{end}}`
    - After the closing `</div>` of the md-block: emit `{{if .ListClose}}{{.ListClose}}{{end}}`
  - Result: consecutive list-item `.md-block` divs are wrapped in `<ul>` or `<ol>`.

- [ ] **T4**: Add CSS for list-item blocks inside list wrappers (`internal/ui/styles.css`) [Stage 1, depends: T3]
  - Files: `internal/ui/styles.css` (modifies)
  - Add rules after the existing `.md-block` rules (~line 360):
    - `.markdown-file-preview > ul, .markdown-file-preview > ol`: Reset margin/padding on the wrapper (`margin: 0; padding: 0; list-style: none;`).
    - `.markdown-file-preview > ul > .md-block, .markdown-file-preview > ol > .md-block`: Tighter vertical padding (`padding-top: 1px; padding-bottom: 1px;`) so items appear as a cohesive group.
    - `.markdown-file-preview > ul > .md-block .md-block-content, .markdown-file-preview > ol > .md-block .md-block-content`: Add `padding-left: 2em` for list indentation.
    - Ensure `<li>` displays as `list-item` with appropriate `list-style-type` (disc for ul, decimal for ol).

### Tests

- [ ] **T5**: Write all tests for per-list-item block splitting (`internal/app/markdown_test.go`) [Stage 1]
  - Files: `internal/app/markdown_test.go` (modifies)
  - Update `TestRenderMarkdownBlocksLineNumbers`: the 2-item list at lines 6-7 now produces 2 blocks (total 4). Assert `blocks[2]`: StartLine=6, HTML contains `item 1`, `ListOpen` contains `<ul`. Assert `blocks[3]`: StartLine=7, HTML contains `item 2`, `ListClose` contains `</ul>`.
  - Add `TestRenderMarkdownBlocksOrderedListWrapper`: Ordered list with `start="5"`, verifies `<ol start="5">` in `ListOpen`. Also test ordered list starting at 1 has `ListOpen == "<ol>"` without `start` attribute.
  - Add `TestRenderMarkdownBlocksNestedList`: Nested list stays inside parent item's HTML; only top-level items become blocks.
  - Add `TestRenderMarkdownBlocksTaskList`: GFM task list items render checkboxes in per-item blocks.
  - Add `TestRenderMarkdownBlocksNonListUnchanged`: Headings, paragraphs, blockquotes have empty `ListOpen`/`ListClose`.
  - Add `TestMarkdownBlocksListItemCommentProjection`: Comment on line 2 of a 3-item list attaches only to the second item's block.
  - Add `TestMarkdownBlocksListItemSelection`: Selection on line 2 of a 3-item list selects only the second item's block.

## Implementation Notes

- **HTML validity**: `<div>` inside `<ul>`/`<ol>` is technically non-standard HTML, but all modern browsers handle it correctly. This is the pragmatic approach that preserves the existing `.md-block` hover/selection/comment machinery.
- **Scope**: Only lists are split. Blockquotes and tables remain as single blocks — splitting those is a separate future enhancement.
- **No changes to `view.go`**: The existing `projectBlockComments()` and selection logic in `updateFileView()` work correctly with per-item blocks. No modifications needed.
- **No changes to `app.go`**: The `select-line` and `add-comment` event handlers are line-range based and work correctly with the new per-item ranges.
- **No changes to JS**: `Element.closest("[data-line]")` naturally finds the innermost match.

## Key Files

- `internal/app/model.go` — MarkdownBlock struct, receives new ListOpen/ListClose fields
- `internal/app/assets.go` — renderMarkdownBlocks(), core splitting logic
- `internal/ui/template.html` — Emits list wrapper tags around per-item blocks
- `internal/ui/styles.css` — CSS for list-item blocks inside wrappers
- `internal/app/markdown_test.go` — Updated and new tests

## Execution Stages

### Stage 1: Full implementation

#### Test Creation Phase
- T-test-5: Write all tests for per-list-item blocks (hmm-test-writer)
  - Regression tests: non-list blocks produce same blocks as before
  - New feature tests (RED): per-item blocks with correct line ranges, ListOpen/ListClose, ordered lists, nested lists, task lists, comment projection, selection projection

#### Implementation Phase (depends on Test Creation Phase)
- T-impl-1: Add ListOpen/ListClose to MarkdownBlock struct and modify renderMarkdownBlocks() to split ast.List nodes (hmm-implement-worker, TDD mode)
  - T1 + T2: model changes and core splitting logic
  - Make RED tests pass (GREEN)
- T-impl-2: Update template and CSS for list wrapper tags (hmm-implement-worker, TDD mode, depends: T-impl-1)
  - T3 + T4: template emits ListOpen/ListClose, CSS styles list-item blocks
- T-verify: Run full test suite (hmm-implement-worker, depends: T-impl-2)
  - `go test ./...`

## Refs

- `docs/research/2026-04-02-list-item-commenting.md`
