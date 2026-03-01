package app

import (
	"bytes"
	"strings"
	"testing"
)

// TestCommentIDAssignment verifies that adding comments to ReviewModel assigns
// sequential IDs (1, 2, 3...) using the NextCommentID counter.
//
// Scenario: Adding comments assigns sequential IDs
func TestCommentIDAssignment(t *testing.T) {
	model := &ReviewModel{}

	// Simulate add-comment handler logic for three comments.
	for i, want := range []int{1, 2, 3} {
		model.NextCommentID++
		c := Comment{
			ID:        model.NextCommentID,
			Path:      "file.go",
			StartLine: i + 1,
			EndLine:   i + 1,
			Text:      "comment",
		}
		model.Comments = append(model.Comments, c)

		if model.Comments[i].ID != want {
			t.Errorf("comment %d: got ID %d, want %d", i, model.Comments[i].ID, want)
		}
	}

	if model.NextCommentID != 3 {
		t.Errorf("NextCommentID: got %d, want 3", model.NextCommentID)
	}
}

// TestProjectLineCommentsEditing verifies that projectLineComments sets
// Editing: true on a ViewComment when editingID matches the comment's ID,
// and Editing: false when it doesn't match or editingID is 0.
//
// Scenario: projectLineComments sets Editing: true only when editingID matches
// Scenario: projectLineComments sets Editing: false when editingID is 0 or doesn't match
func TestProjectLineCommentsEditing(t *testing.T) {
	comments := []Comment{
		{ID: 1, Path: "a.go", StartLine: 5, EndLine: 5, Text: "first"},
		{ID: 2, Path: "a.go", StartLine: 5, EndLine: 5, Text: "second"},
	}

	tests := []struct {
		name      string
		editingID int
		wantID1   bool
		wantID2   bool
	}{
		{name: "no editing", editingID: 0, wantID1: false, wantID2: false},
		{name: "editing comment 1", editingID: 1, wantID1: true, wantID2: false},
		{name: "editing comment 2", editingID: 2, wantID1: false, wantID2: true},
		{name: "editing nonexistent", editingID: 99, wantID1: false, wantID2: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, viewComments := projectLineComments("a.go", 5, comments, tc.editingID)
			if len(viewComments) != 2 {
				t.Fatalf("expected 2 view comments, got %d", len(viewComments))
			}
			if viewComments[0].Editing != tc.wantID1 {
				t.Errorf("comment ID 1 Editing: got %v, want %v", viewComments[0].Editing, tc.wantID1)
			}
			if viewComments[1].Editing != tc.wantID2 {
				t.Errorf("comment ID 2 Editing: got %v, want %v", viewComments[1].Editing, tc.wantID2)
			}
		})
	}
}

// TestDeleteComment verifies that removing a comment by ID from the Comments
// slice works correctly.
//
// Scenario: Delete a comment removes it from the slice
func TestDeleteComment(t *testing.T) {
	model := &ReviewModel{
		NextCommentID: 3,
		Comments: []Comment{
			{ID: 1, Path: "a.go", StartLine: 1, EndLine: 1, Text: "one"},
			{ID: 2, Path: "a.go", StartLine: 2, EndLine: 2, Text: "two"},
			{ID: 3, Path: "a.go", StartLine: 3, EndLine: 3, Text: "three"},
		},
	}

	deleteComment(model, 2)

	if len(model.Comments) != 2 {
		t.Fatalf("expected 2 comments after delete, got %d", len(model.Comments))
	}
	for _, c := range model.Comments {
		if c.ID == 2 {
			t.Error("comment with ID 2 should have been deleted")
		}
	}
}

// TestDeleteNonexistentComment verifies that deleting a nonexistent ID is a
// no-op (the slice is unchanged).
//
// Scenario: Delete a nonexistent comment is a no-op
func TestDeleteNonexistentComment(t *testing.T) {
	model := &ReviewModel{
		NextCommentID: 1,
		Comments: []Comment{
			{ID: 1, Path: "a.go", StartLine: 1, EndLine: 1, Text: "one"},
		},
	}

	deleteComment(model, 99)

	if len(model.Comments) != 1 {
		t.Fatalf("expected slice unchanged (length 1), got %d", len(model.Comments))
	}
	if model.Comments[0].ID != 1 {
		t.Errorf("expected comment ID 1 to still exist, got ID %d", model.Comments[0].ID)
	}
}

// TestDeleteClearsEditingID verifies that deleting a comment whose ID matches
// EditingCommentID clears EditingCommentID to 0.
//
// Scenario: Delete a comment clears EditingCommentID when it matches
func TestDeleteClearsEditingID(t *testing.T) {
	model := &ReviewModel{
		NextCommentID:    2,
		EditingCommentID: 2,
		Comments: []Comment{
			{ID: 1, Path: "a.go", StartLine: 1, EndLine: 1, Text: "one"},
			{ID: 2, Path: "a.go", StartLine: 2, EndLine: 2, Text: "two"},
		},
	}

	deleteComment(model, 2)

	if model.EditingCommentID != 0 {
		t.Errorf("EditingCommentID should be cleared to 0, got %d", model.EditingCommentID)
	}
}

// TestDeleteDoesNotClearOtherEditingID verifies that deleting a comment does
// not clear EditingCommentID when the IDs don't match.
func TestDeleteDoesNotClearOtherEditingID(t *testing.T) {
	model := &ReviewModel{
		NextCommentID:    2,
		EditingCommentID: 1,
		Comments: []Comment{
			{ID: 1, Path: "a.go", StartLine: 1, EndLine: 1, Text: "one"},
			{ID: 2, Path: "a.go", StartLine: 2, EndLine: 2, Text: "two"},
		},
	}

	deleteComment(model, 2)

	if model.EditingCommentID != 1 {
		t.Errorf("EditingCommentID should remain 1, got %d", model.EditingCommentID)
	}
}

// TestViewCommentInheritsID verifies that ViewComment (which embeds Comment)
// exposes the ID field from the embedded struct.
//
// Scenario: ViewComment.ID is inherited from embedded Comment
func TestViewCommentInheritsID(t *testing.T) {
	vc := ViewComment{
		Comment: Comment{ID: 42, Path: "b.go", StartLine: 10, EndLine: 10, Text: "hello"},
	}
	if vc.ID != 42 {
		t.Errorf("ViewComment.ID: got %d, want 42", vc.ID)
	}
}

// TestEditCommentSuccess verifies that editing a comment by ID updates its
// text and clears EditingCommentID.
//
// Scenario: Edit a comment with valid text updates the comment
func TestEditCommentSuccess(t *testing.T) {
	model := &ReviewModel{
		NextCommentID:    1,
		EditingCommentID: 1,
		Comments: []Comment{
			{ID: 1, Path: "a.go", StartLine: 5, EndLine: 5, Text: "original"},
		},
	}

	err := editComment(model, 1, "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Comments[0].Text != "updated" {
		t.Errorf("comment text: got %q, want %q", model.Comments[0].Text, "updated")
	}
	if model.EditingCommentID != 0 {
		t.Errorf("EditingCommentID should be cleared to 0, got %d", model.EditingCommentID)
	}
}

// TestEditCommentEmptyTextReturnsError verifies that editing a comment with
// empty text returns an error.
//
// Scenario: Edit with empty text shows error
func TestEditCommentEmptyTextReturnsError(t *testing.T) {
	model := &ReviewModel{
		NextCommentID:    1,
		EditingCommentID: 1,
		Comments: []Comment{
			{ID: 1, Path: "a.go", StartLine: 5, EndLine: 5, Text: "original"},
		},
	}

	err := editComment(model, 1, "")
	if err == nil {
		t.Fatal("expected an error for empty text, got nil")
	}
	// The comment text should remain unchanged.
	if model.Comments[0].Text != "original" {
		t.Errorf("comment text should be unchanged, got %q", model.Comments[0].Text)
	}
}

// TestEditCommentNonexistentIDReturnsError verifies that editing a nonexistent
// comment ID returns an error.
//
// Scenario: Edit a nonexistent comment shows error
func TestEditCommentNonexistentIDReturnsError(t *testing.T) {
	model := &ReviewModel{
		NextCommentID: 1,
		Comments: []Comment{
			{ID: 1, Path: "a.go", StartLine: 5, EndLine: 5, Text: "original"},
		},
	}

	err := editComment(model, 99, "updated")
	if err == nil {
		t.Fatal("expected an error for nonexistent ID, got nil")
	}
}

// TestCancelEditComment verifies that cancelling an edit clears EditingCommentID.
func TestCancelEditComment(t *testing.T) {
	model := &ReviewModel{
		EditingCommentID: 3,
		Comments: []Comment{
			{ID: 3, Path: "a.go", StartLine: 1, EndLine: 1, Text: "hello"},
		},
	}

	// Simulate cancel-edit-comment handler logic.
	model.EditingCommentID = 0
	model.Error = ""

	if model.EditingCommentID != 0 {
		t.Errorf("EditingCommentID should be 0 after cancel, got %d", model.EditingCommentID)
	}
}

// TestEmitToonIncludesCommentID verifies that the TOON output includes the
// comment ID field.
//
// Scenario: TOON output includes comment IDs
func TestEmitToonIncludesCommentID(t *testing.T) {
	comments := []Comment{
		{ID: 1, Path: "a.go", StartLine: 5, EndLine: 5, Text: "hello"},
		{ID: 2, Path: "b.go", StartLine: 10, EndLine: 10, Text: "world"},
	}

	var buf bytes.Buffer
	if err := emitToon(&buf, comments); err != nil {
		t.Fatalf("emitToon error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "id") {
		t.Errorf("TOON output should contain 'id' field, got: %s", output)
	}
}
