package app

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jfyne/live"
)

func TestHTTPRenderIncludesThemeClass(t *testing.T) {
	model := &ReviewModel{
		Files:                []File{{Path: "a.go", PathSlash: "a.go", Lines: []string{"package main"}}},
		SelectedPath:         "a.go",
		Mode:                 ModeFile,
		RenderFile:           true,
		RenderComments:       true,
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)
	if !strings.Contains(html, `<body class="theme-dark">`) {
		t.Fatalf("expected rendered html to include theme class for chroma css, got: %q", html)
	}
}

func TestHTTPRenderFileModeCommentFormAutofocus(t *testing.T) {
	model := &ReviewModel{
		Files: []File{{
			Path:      "a.go",
			PathSlash: "a.go",
			Lines:     []string{"package main", "func main() {}"},
		}},
		SelectedPath:         "a.go",
		SelectionStart:       2,
		SelectionEnd:         2,
		Mode:                 ModeFile,
		RenderFile:           true,
		RenderComments:       true,
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)
	if !strings.Contains(html, `<textarea name="comment" placeholder="Leave a comment..." autofocus></textarea>`) {
		t.Fatalf("expected comment textarea autofocus in file mode, got: %q", html)
	}
}

func TestHTTPRenderDiffModeCommentFormAutofocus(t *testing.T) {
	model := &ReviewModel{
		DiffFiles: []DiffFile{{
			Path: "a.go",
			Hunks: []DiffHunk{{
				OldStart: 1,
				OldCount: 1,
				NewStart: 1,
				NewCount: 1,
				Lines: []DiffLine{
					{Kind: DiffContext, OldLine: 1, NewLine: 1, Text: "package main"},
				},
			}},
		}},
		SelectedPath:         "a.go",
		SelectionStart:       1,
		SelectionEnd:         1,
		Mode:                 ModeDiff,
		RenderFile:           true,
		RenderComments:       true,
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(diffFilesAsFiles(model.DiffFiles), model.SelectedPath)

	html := renderReviewHTML(t, model)
	if !strings.Contains(html, `<textarea name="comment" placeholder="Leave a comment..." autofocus></textarea>`) {
		t.Fatalf("expected comment textarea autofocus in diff mode, got: %q", html)
	}
}

// buildCommentModel returns a ReviewModel in file mode with a single comment
// (ID 1, path "test.go", line 1, text "hello") ready for rendering.
// Callers may mutate the returned model before calling renderReviewHTML.
func buildCommentModel() *ReviewModel {
	return &ReviewModel{
		Files: []File{{
			Path:      "test.go",
			PathSlash: "test.go",
			Lines:     []string{"package main"},
		}},
		SelectedPath:         "test.go",
		Mode:                 ModeFile,
		RenderFile:           true,
		RenderComments:       true,
		NextCommentID:        1,
		Comments: []Comment{
			{ID: 1, Path: "test.go", StartLine: 1, EndLine: 1, Text: "hello"},
		},
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
}

// TestRenderCommentEditDeleteButtons verifies that each rendered comment
// displays an edit button (live-click="start-edit-comment") and a delete
// button (live-click="delete-comment"), both carrying the comment's ID.
//
// Scenario: Edit and delete buttons visible on comments
func TestRenderCommentEditDeleteButtons(t *testing.T) {
	model := buildCommentModel()
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)

	if !strings.Contains(html, `live-click="start-edit-comment"`) {
		t.Errorf("expected start-edit-comment button in rendered HTML, got: %q", html)
	}
	if !strings.Contains(html, `live-click="delete-comment"`) {
		t.Errorf("expected delete-comment button in rendered HTML, got: %q", html)
	}
	if !strings.Contains(html, `live-value-id="1"`) {
		t.Errorf("expected live-value-id=\"1\" on comment action buttons, got: %q", html)
	}
}

// TestRenderCommentEditForm verifies that when EditingCommentID matches a
// comment's ID the rendered HTML shows an edit form pre-filled with the
// comment text, and hides the edit/delete action buttons for that comment.
//
// Scenario: Edit form appears with pre-filled text
// Scenario: Buttons hidden during editing
func TestRenderCommentEditForm(t *testing.T) {
	model := buildCommentModel()
	model.EditingCommentID = 1
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)

	if !strings.Contains(html, `live-submit="edit-comment"`) {
		t.Errorf("expected edit-comment form in rendered HTML, got: %q", html)
	}
	if !strings.Contains(html, `<textarea`) {
		t.Errorf("expected textarea in edit form, got: %q", html)
	}
	if !strings.Contains(html, `hello`) {
		t.Errorf("expected comment text \"hello\" pre-filled in textarea, got: %q", html)
	}
	if !strings.Contains(html, `name="id"`) {
		t.Errorf("expected hidden id input (name=\"id\") in edit form, got: %q", html)
	}
	if !strings.Contains(html, `value="1"`) {
		t.Errorf("expected value=\"1\" on hidden id input, got: %q", html)
	}
	// Edit and delete action buttons should be hidden while editing.
	if strings.Contains(html, `live-click="start-edit-comment"`) {
		t.Errorf("start-edit-comment button should be hidden when comment is being edited, got: %q", html)
	}
	if strings.Contains(html, `live-click="delete-comment"`) {
		t.Errorf("delete-comment button should be hidden when comment is being edited, got: %q", html)
	}
}

// TestRenderCommentNotEditing verifies that when EditingCommentID is 0 (no
// comment being edited) the edit/delete action buttons are rendered and no
// edit form is present.
//
// Scenario: Edit and delete buttons visible on comments (not editing state)
func TestRenderCommentNotEditing(t *testing.T) {
	model := buildCommentModel()
	model.EditingCommentID = 0
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)

	if !strings.Contains(html, `live-click="start-edit-comment"`) {
		t.Errorf("expected start-edit-comment button when not editing, got: %q", html)
	}
	if !strings.Contains(html, `live-click="delete-comment"`) {
		t.Errorf("expected delete-comment button when not editing, got: %q", html)
	}
	if strings.Contains(html, `live-submit="edit-comment"`) {
		t.Errorf("edit form should not be present when not editing, got: %q", html)
	}
}

func renderReviewHTML(t *testing.T, model *ReviewModel) string {
	t.Helper()

	updateView(model)

	rs := &ReviewServer{Model: model, DoneCh: make(chan struct{})}
	h := buildLiveHandler(rs)
	out, err := h.RenderHandler(context.Background(), &live.RenderContext{Assigns: model})
	if err != nil {
		t.Fatalf("render handler failed: %v", err)
	}
	b, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("read render output failed: %v", err)
	}
	return string(b)
}
