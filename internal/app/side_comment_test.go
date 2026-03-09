package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestProjectLineCommentsSideAware verifies that projectLineComments, once it
// gains a side parameter, filters comments by their Side field.
//
// Scenario: side="" matches only comments with c.Side=="" (backward compat)
// Scenario: side="" does NOT match comments with c.Side="old"
// Scenario: side="old" matches comments with c.Side="old"
// Scenario: side="old" does NOT match comments with c.Side==""
func TestProjectLineCommentsSideAware(t *testing.T) {
	newSideComment := Comment{
		ID:        1,
		Path:      "file.go",
		StartLine: 5,
		EndLine:   5,
		Text:      "new-side comment",
		Side:      "",
	}
	oldSideComment := Comment{
		ID:        2,
		Path:      "file.go",
		StartLine: 5,
		EndLine:   5,
		Text:      "old-side comment",
		Side:      "old",
	}
	allComments := []Comment{newSideComment, oldSideComment}

	tests := []struct {
		name            string
		side            string
		wantCommented   bool
		wantCount       int
		wantCommentText string
	}{
		{
			name:            "side='' matches new-side comment only",
			side:            "",
			wantCommented:   true,
			wantCount:       1,
			wantCommentText: "new-side comment",
		},
		{
			name:          "side='' does not match old-side comment",
			side:          "",
			wantCommented: true,
			wantCount:     1,
			// Verified by checking the single result is NOT the old-side one.
			wantCommentText: "new-side comment",
		},
		{
			name:            "side='old' matches old-side comment only",
			side:            "old",
			wantCommented:   true,
			wantCount:       1,
			wantCommentText: "old-side comment",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			commented, viewComments := projectLineComments("file.go", 5, allComments, 0, tc.side)
			if commented != tc.wantCommented {
				t.Errorf("commented: got %v, want %v", commented, tc.wantCommented)
			}
			if len(viewComments) != tc.wantCount {
				t.Fatalf("len(viewComments): got %d, want %d", len(viewComments), tc.wantCount)
			}
			if viewComments[0].Text != tc.wantCommentText {
				t.Errorf("comment text: got %q, want %q", viewComments[0].Text, tc.wantCommentText)
			}
		})
	}

	// Explicit sub-test: side="old" must NOT return the new-side comment.
	t.Run("side='old' does not match new-side comment", func(t *testing.T) {
		_, viewComments := projectLineComments("file.go", 5, allComments, 0, "old")
		for _, vc := range viewComments {
			if vc.Side != "old" {
				t.Errorf("unexpected comment with Side=%q returned for side='old' query", vc.Side)
			}
		}
	})

	// Explicit sub-test: side="" must NOT return the old-side comment.
	t.Run("side='' does not match old-side comment", func(t *testing.T) {
		_, viewComments := projectLineComments("file.go", 5, allComments, 0, "")
		for _, vc := range viewComments {
			if vc.Side != "" {
				t.Errorf("unexpected comment with Side=%q returned for side='' query", vc.Side)
			}
		}
	})
}

// TestProjectLineCommentsSideAwareNotCommented verifies that when side="old" is
// requested but only a new-side comment exists on the line, commented is false.
//
// Scenario: side="old" does NOT set commented when only new-side comment exists
func TestProjectLineCommentsSideAwareNotCommented(t *testing.T) {
	comments := []Comment{
		{ID: 1, Path: "file.go", StartLine: 3, EndLine: 3, Text: "new only", Side: ""},
	}

	commented, viewComments := projectLineComments("file.go", 3, comments, 0, "old")
	if commented {
		t.Error("commented should be false when no old-side comment exists on the line")
	}
	if len(viewComments) != 0 {
		t.Errorf("viewComments: got %d, want 0", len(viewComments))
	}
}

// TestProjectLineCommentsSideAwareDifferentInputs verifies different line/path
// combos to prevent the implementation from returning a hardcoded value.
func TestProjectLineCommentsSideAwareDifferentInputs(t *testing.T) {
	comments := []Comment{
		{ID: 10, Path: "alpha.go", StartLine: 1, EndLine: 1, Text: "alpha line1 new", Side: ""},
		{ID: 11, Path: "alpha.go", StartLine: 2, EndLine: 2, Text: "alpha line2 old", Side: "old"},
		{ID: 12, Path: "beta.go", StartLine: 1, EndLine: 1, Text: "beta line1 old", Side: "old"},
	}

	// alpha.go line 1 new-side: expects 1 comment with ID 10
	_, got := projectLineComments("alpha.go", 1, comments, 0, "")
	if len(got) != 1 || got[0].ID != 10 {
		t.Errorf("alpha.go line 1 side='': got %v, want [{ID:10}]", got)
	}

	// alpha.go line 2 old-side: expects 1 comment with ID 11
	_, got = projectLineComments("alpha.go", 2, comments, 0, "old")
	if len(got) != 1 || got[0].ID != 11 {
		t.Errorf("alpha.go line 2 side='old': got %v, want [{ID:11}]", got)
	}

	// beta.go line 1 new-side: expects 0 comments (only old-side comment exists)
	_, got = projectLineComments("beta.go", 1, comments, 0, "")
	if len(got) != 0 {
		t.Errorf("beta.go line 1 side='': got %d comments, want 0", len(got))
	}

	// beta.go line 1 old-side: expects 1 comment with ID 12
	_, got = projectLineComments("beta.go", 1, comments, 0, "old")
	if len(got) != 1 || got[0].ID != 12 {
		t.Errorf("beta.go line 1 side='old': got %v, want [{ID:12}]", got)
	}
}

// TestCommentSideFieldOmitEmpty verifies that a Comment with Side="" marshals
// to JSON without the "side" key (omitempty behavior).
//
// Scenario: Comment with Side="" marshals without "side" key in JSON
func TestCommentSideFieldOmitEmpty(t *testing.T) {
	c := Comment{
		ID:        1,
		Path:      "a.go",
		StartLine: 1,
		EndLine:   1,
		Text:      "hello",
		Side:      "",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	got := string(data)
	if strings.Contains(got, `"side"`) {
		t.Errorf("JSON should not contain 'side' key when Side is empty, got: %s", got)
	}
	// Verify other fields are present.
	if !strings.Contains(got, `"id":1`) {
		t.Errorf("JSON missing 'id' field, got: %s", got)
	}
	if !strings.Contains(got, `"path":"a.go"`) {
		t.Errorf("JSON missing 'path' field, got: %s", got)
	}
}

// TestCommentSideFieldPresent verifies that a Comment with Side="old" marshals
// to JSON with "side":"old".
//
// Scenario: Comment with Side="old" marshals with "side":"old" in JSON
func TestCommentSideFieldPresent(t *testing.T) {
	c := Comment{
		ID:        2,
		Path:      "b.go",
		StartLine: 5,
		EndLine:   5,
		Text:      "old side comment",
		Side:      "old",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	got := string(data)
	if !strings.Contains(got, `"side":"old"`) {
		t.Errorf("JSON should contain '\"side\":\"old\"', got: %s", got)
	}
	// Verify other fields are present.
	if !strings.Contains(got, `"id":2`) {
		t.Errorf("JSON missing 'id' field, got: %s", got)
	}
	if !strings.Contains(got, `"path":"b.go"`) {
		t.Errorf("JSON missing 'path' field, got: %s", got)
	}
}
