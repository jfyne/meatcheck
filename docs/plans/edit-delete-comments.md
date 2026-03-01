# Implementation Plan: Edit and Delete Existing Comments

Add the ability for users to edit and delete comments they have already placed on a document or diff. Currently, once a comment is submitted, it cannot be modified or removed.

## Context

**Key Files**:
- `internal/app/model.go` - Defines `Comment`, `ViewComment`, `ReviewModel` structs
- `internal/app/app.go` - LiveView event handlers (`add-comment`, `cancel-comment`, etc.)
- `internal/app/view.go` - View projection logic (`projectLineComments`, `buildViewLines`, `buildViewDiff`)
- `internal/ui/template.html` - HTML template with `commentThread` sub-template
- `internal/ui/styles.css` - All CSS styles
- `internal/app/io_helpers.go` - TOON output serialization of comments

**Architectural Notes**:
- Server-side LiveView (github.com/jfyne/live) — all state lives on `ReviewModel`, events are handled server-side, and the full template re-renders on each event.
- Event handler pattern: `h.HandleEvent("name", func(ctx, socket, params) (model, error))`.
- Form data accessed via `p.String("name")` / `p.Int("name")` matching HTML form field `name` attributes.
- Comments are currently identified only by slice position — no stable identity exists.

**Functional Requirements** (EARS notation):
- When the user clicks the edit button on a comment, the system shall replace the comment body with an editable textarea pre-filled with the comment text.
- When the user submits the edit form, the system shall update the comment text and return to display mode.
- When the user clicks cancel during editing, the system shall discard changes and return to display mode.
- When the user clicks the delete button on a comment, the system shall remove the comment from the review.
- While a comment is being edited, the system shall hide the edit/delete buttons for that comment.
- If the user submits an edit with empty text, the system shall display an error and keep the edit form open.

## Batch Size

| Metric | Count | Rating |
|--------|-------|--------|
| Tasks | 7 | Medium |
| Files | 6 | Small |
| Stages | 2 | Small |

**Overall: Medium**

## Execution Stages

### Stage 1: Backend — Model, View Projection, Event Handlers

#### Test Creation Phase (parallel)
- T-test-1: Write tests for comment ID assignment, edit, delete, and editing state projection (hmm-test-writer)
  - Regression tests: existing `buildViewDiff` with comments (view_diff_test.go)
  - New feature tests (RED): comment ID auto-increment, edit-by-ID, delete-by-ID, projectLineComments editing flag

#### Implementation Phase (sequential, depends on Test Creation Phase)
- T-impl-1: Add ID to Comment, NextCommentID/EditingCommentID to ReviewModel, Editing to ViewComment (hmm-implement-worker, TDD mode)
- T-impl-2: Add event handlers and update view projection (hmm-implement-worker, TDD mode)
  - Depends on T-impl-1 (references struct fields created by T-impl-1)

### Stage 2: Frontend — Template and CSS (depends on Stage 1)

#### Test Creation Phase (parallel)
- T-test-2: Write render tests for edit/delete button presence and edit form rendering (hmm-test-writer)
  - New feature tests (RED): rendered HTML contains edit/delete buttons, edit form appears when EditingCommentID is set

#### Implementation Phase (parallel, depends on Test Creation Phase)
- T-impl-3: Update commentThread template and add CSS styles (hmm-implement-worker, TDD mode)

## Task List

### Backend: Model Changes

- [x] Add `ID` to `Comment`, `NextCommentID`/`EditingCommentID` to `ReviewModel`, `Editing` to `ViewComment` (`internal/app/model.go`) [Stage 1]
  - Files: `internal/app/model.go` (modifies)
  - Add `ID int \`json:"id"\`` as first field on `Comment`
  - Add `NextCommentID int` and `EditingCommentID int` to `ReviewModel` (near `Comments` field)
  - Add `Editing bool` to `ViewComment`
  - Zero values are safe defaults (ID 0 = unset, EditingCommentID 0 = not editing)

### Backend: View Projection

- [x] Thread `editingID` through view projection functions (`internal/app/view.go`) [Stage 1]
  - Files: `internal/app/view.go` (modifies)
  - Add `editingID int` parameter to: `projectLineComments`, `buildSingleViewLine`, `buildViewLines`, `buildViewLinesWithRanges`, `buildViewDiff`
  - In `projectLineComments`, set `Editing: c.ID == editingID` on each `ViewComment`
  - In `updateFileView` and `updateDiffView`, pass `model.EditingCommentID` to the builders

### Backend: Event Handlers

- [x] Assign ID on add-comment, add edit/delete/start-edit/cancel-edit handlers (`internal/app/app.go`) [Stage 1]
  - Files: `internal/app/app.go` (modifies)
  - **add-comment** (modify existing): increment `model.NextCommentID++` and set `ID: model.NextCommentID` on the new comment
  - **start-edit-comment** (new): set `model.EditingCommentID = p.Int("id")`, call `updateView`
  - **edit-comment** (new): find comment by `p.Int("id")`, update `Text` to `p.String("comment")`, clear `EditingCommentID`, call `updateView`. Error if text empty or ID not found.
  - **delete-comment** (new): find comment by `p.Int("id")`, remove from slice. Clear `EditingCommentID` if it matches. Idempotent (no error on missing ID).
  - **cancel-edit-comment** (new): set `model.EditingCommentID = 0`, call `updateView`
  - Note: edit-comment reads `p.String("comment")` (not "text") to match the textarea `name="comment"` attribute in the template

### Backend: Fix Existing Tests

- [x] Update existing tests for new function signatures (`internal/app/view_diff_test.go`) [Stage 1]
  - Files: `internal/app/view_diff_test.go` (modifies)
  - Add `ID: 1` to `Comment` literal
  - Add `editingID` (0) argument to `buildViewDiff` call
  - Verify test still passes with no behavioral change

### Frontend: Template

- [x] Add edit/delete buttons and edit form to `commentThread` template (`internal/ui/template.html`) [Stage 2]
  - Files: `internal/ui/template.html` (modifies)
  - In `commentThread` template:
    - Restructure `.line-comment-meta` to use flex layout with a `<span>` for the path label and a `<span class="line-comment-actions">` for buttons
    - Add Edit button (`live-click="start-edit-comment" live-value-id="{{.ID}}"`) using pencil character &#9998;
    - Add Delete button (`live-click="delete-comment" live-value-id="{{.ID}}"`) using &times; character
    - Hide buttons when `{{.Editing}}` is true
    - When `{{.Editing}}` is true, show a `<form live-submit="edit-comment">` with hidden input for ID, textarea pre-filled with `.Text`, and Save/Cancel buttons
    - Add `autofocus` to edit textarea
  - In JS cancel-button handler (line 192): change selector to `target.matches('button[live-click="cancel-comment"], button[live-click="cancel-edit-comment"]')`
  - Ctrl+Enter already works automatically (matches any `textarea[name="comment"]`)

### Frontend: CSS

- [x] Add styles for comment action buttons and edit form (`internal/ui/styles.css`) [Stage 2]
  - Files: `internal/ui/styles.css` (modifies)
  - Update `.line-comment-meta` to `display: flex; align-items: center; justify-content: space-between`
  - `.line-comment-actions` — inline-flex container with small gap
  - `.comment-action-btn` — transparent background, muted color, subtle hover effect
  - `.comment-action-btn.delete:hover` — warn color on hover
  - `.edit-comment-form` — border, padding, smaller min-height on textarea

### Backend: New Tests

- [x] Write tests for comment edit/delete logic (`internal/app/comment_test.go`) [Stage 1]
  - Files: `internal/app/comment_test.go` (creates)
  - Test cases:
    - Adding comments assigns sequential IDs (1, 2, 3...)
    - `projectLineComments` sets `Editing: true` only when `editingID` matches comment ID
    - `projectLineComments` sets `Editing: false` when `editingID` is 0 or doesn't match
    - Deleting a comment removes it from the slice
    - Deleting a nonexistent ID is a no-op
    - Deleting a comment clears `EditingCommentID` when it matches the deleted comment's ID
    - Editing a comment with empty text returns an error
    - Editing a comment with a nonexistent ID returns an error
    - ViewComment.ID is inherited from embedded Comment

## Acceptance Criteria

~~~gherkin
Feature: Edit and delete existing comments

  Scenario: Edit a comment
    Given a comment exists on line 5 with text "original"
    When the user clicks the edit button on that comment
    Then a textarea appears pre-filled with "original"
    When the user changes the text to "updated" and submits
    Then the comment displays "updated"

  Scenario: Cancel editing a comment
    Given a comment is being edited
    When the user clicks Cancel
    Then the edit form disappears
    And the original comment text is displayed unchanged

  Scenario: Edit with empty text shows error
    Given a comment is being edited
    When the user clears the textarea and submits
    Then an error message "comment text is required" is displayed
    And the edit form remains open

  Scenario: Delete a comment
    Given a comment exists on line 10
    When the user clicks the delete button on that comment
    Then the comment is removed from the display
    And the comment is not included in the final TOON output

  Scenario: Only one comment editable at a time
    Given comment A and comment B exist
    When the user clicks edit on comment A
    And then clicks edit on comment B
    Then only comment B shows the edit form
    And comment A returns to display mode

  Scenario: Edit and delete buttons visible on comments
    Given a comment exists
    Then edit and delete buttons are visible in the comment header
    When the comment is being edited
    Then the edit and delete buttons are hidden

  Scenario: Delete a nonexistent comment is a no-op
    Given a comment with ID 1 exists
    When a delete event is sent for ID 99
    Then the comment with ID 1 still exists
    And no error is displayed

  Scenario: Edit a nonexistent comment shows error
    Given a comment with ID 1 exists
    When an edit event is sent for ID 99 with text "updated"
    Then an error message "comment not found" is displayed

  Scenario: TOON output includes comment IDs
    Given comments have been added
    When the user clicks Finish
    Then the TOON output includes an "id" field for each comment
~~~

**Source**: Generated from plan context

## Implementation Notes

- **Comment ID strategy**: Simple auto-incrementing counter on `ReviewModel.NextCommentID`. IDs start at 1 (zero reserved for "none"). IDs are never reused after deletion. Sufficient for single-session, single-user tool.
- **TOON output**: The `ID` field will appear in the serialized JSON output since it has a `json:"id"` tag. This is acceptable and potentially useful for downstream consumers.
- **Concurrency**: `ReviewModel` has no mutex, which is an existing pattern. Edit/delete add more mutation paths but the tool is single-user, so this is acceptable.
- **Form field naming**: The edit-comment handler reads `p.String("comment")` to match the textarea `name="comment"`. The comment ID is sent via hidden input `name="id"` read as `p.Int("id")`.

## Refs

- No associated GitHub issue
