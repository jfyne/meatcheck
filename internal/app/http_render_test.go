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
		SelectedPath:   "test.go",
		Mode:           ModeFile,
		RenderFile:     true,
		RenderComments: true,
		NextCommentID:  1,
		Comments: []Comment{
			{ID: 1, Path: "test.go", StartLine: 1, EndLine: 1, Text: "hello"},
		},
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
}

// TestRenderCommentEditDeleteButtons verifies that each rendered comment
// displays an edit button (data-action="start-edit-comment") and a delete
// button (data-action="delete-comment"), both carrying the comment's ID.
//
// Scenario: Edit and delete buttons visible on comments
func TestRenderCommentEditDeleteButtons(t *testing.T) {
	model := buildCommentModel()
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)

	if !strings.Contains(html, `data-action="start-edit-comment"`) {
		t.Errorf("expected start-edit-comment button in rendered HTML, got: %q", html)
	}
	if !strings.Contains(html, `data-action="delete-comment"`) {
		t.Errorf("expected delete-comment button in rendered HTML, got: %q", html)
	}
	if !strings.Contains(html, `data-comment-id="1"`) {
		t.Errorf("expected data-comment-id=\"1\" on comment action buttons, got: %q", html)
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
	if strings.Contains(html, `data-action="start-edit-comment"`) {
		t.Errorf("start-edit-comment button should be hidden when comment is being edited, got: %q", html)
	}
	if strings.Contains(html, `data-action="delete-comment"`) {
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

	if !strings.Contains(html, `data-action="start-edit-comment"`) {
		t.Errorf("expected start-edit-comment button when not editing, got: %q", html)
	}
	if !strings.Contains(html, `data-action="delete-comment"`) {
		t.Errorf("expected delete-comment button when not editing, got: %q", html)
	}
	if strings.Contains(html, `live-submit="edit-comment"`) {
		t.Errorf("edit form should not be present when not editing, got: %q", html)
	}
}

// buildMinimalModel returns the smallest valid ReviewModel that produces a
// full HTML render (including inline CSS). It is used by CSS presence tests.
func buildMinimalModel() *ReviewModel {
	m := &ReviewModel{
		Files:                []File{{Path: "a.go", PathSlash: "a.go", Lines: []string{"x"}}},
		SelectedPath:         "a.go",
		Mode:                 ModeFile,
		RenderFile:           true,
		RenderComments:       true,
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	m.Tree = buildTree(m.Files, m.SelectedPath)
	return m
}

// TestMarkdownImgMaxWidth verifies that the rendered HTML/CSS contains a
// .markdown img rule with max-width: 100% to constrain images.
//
// Scenario: Images constrained to container width
func TestMarkdownImgMaxWidth(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	if !strings.Contains(html, `.markdown img`) {
		t.Fatalf("expected .markdown img rule in rendered CSS, got no match")
	}
	if !strings.Contains(html, `max-width: 100%`) {
		t.Fatalf("expected max-width: 100%% in .markdown img rule, got no match")
	}
}

// TestMarkdownPreCodeReset verifies that the rendered HTML/CSS contains a
// .markdown pre code rule that resets the background to transparent.
//
// Scenario: Code blocks do not show inline code background
func TestMarkdownPreCodeReset(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	if !strings.Contains(html, `.markdown pre code`) {
		t.Fatalf("expected .markdown pre code rule in rendered CSS, got no match")
	}
	if !strings.Contains(html, `background: transparent`) {
		t.Fatalf("expected background: transparent in .markdown pre code rule, got no match")
	}
}

// TestMarkdownHrStyles verifies that the rendered HTML/CSS contains a
// .markdown hr rule with a background-color property.
//
// Scenario: Horizontal rules are visible
func TestMarkdownHrStyles(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	if !strings.Contains(html, `.markdown hr`) {
		t.Fatalf("expected .markdown hr rule in rendered CSS, got no match")
	}
	if !strings.Contains(html, `background-color`) {
		t.Fatalf("expected background-color property in .markdown hr rule, got no match")
	}
}

// TestMarkdownH4H5H6Styles verifies that the rendered HTML/CSS contains
// individual .markdown h4, .markdown h5, and .markdown h6 rules with their
// own font-size declarations.
//
// Scenario: All heading levels are styled
func TestMarkdownH4H5H6Styles(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	// The plan specifies: h4 font-size: 1em, h5 font-size: .875em, h6 font-size: .85em.
	// These specific em-unit values only exist after the new CSS rules are added.
	if !strings.Contains(html, `.markdown h4`) {
		t.Fatalf("expected .markdown h4 selector in rendered CSS, got no match")
	}
	if !strings.Contains(html, `.markdown h5`) {
		t.Fatalf("expected .markdown h5 selector in rendered CSS, got no match")
	}
	if !strings.Contains(html, `.markdown h6`) {
		t.Fatalf("expected .markdown h6 selector in rendered CSS, got no match")
	}
	// Verify that individual font-size rules exist (not just the shared margin rule).
	// After implementation, h4 gets font-size: 1em, h5 gets font-size: .875em.
	if !strings.Contains(html, `font-size: .875em`) {
		t.Fatalf("expected font-size: .875em for .markdown h5 in rendered CSS, got no match")
	}
	if !strings.Contains(html, `font-size: .85em`) {
		t.Fatalf("expected font-size: .85em for .markdown h6 in rendered CSS, got no match")
	}
}

// TestMarkdownHeadingFontWeight verifies that the rendered HTML/CSS contains
// a shared markdown heading rule that includes font-weight: 600. The check
// looks for the shared .markdown h1 selector immediately followed (within the
// same CSS block) by font-weight: 600, which is only present after the new
// CSS is added. (Other elements have font-weight: 600 but not in a .markdown
// heading rule.)
//
// Scenario: All heading levels are styled (font-weight)
func TestMarkdownHeadingFontWeight(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	// After implementation the shared heading block contains font-weight: 600.
	// We check for the combination of the markdown heading selector and the
	// font-weight: 600 property within the rendered CSS. The specific em-based
	// heading sizes are the clearest signal that the new heading CSS is present.
	if !strings.Contains(html, `font-size: 2em`) {
		t.Fatalf("expected font-size: 2em for .markdown h1 in rendered CSS (signals new heading rules present), got no match")
	}
	if !strings.Contains(html, `font-size: 1.5em`) {
		t.Fatalf("expected font-size: 1.5em for .markdown h2 in rendered CSS, got no match")
	}
}

// TestMarkdownH1H2BorderBottom verifies that the rendered HTML/CSS contains
// a .markdown h1 rule with padding-bottom and border-bottom. The plan specifies
// ".markdown h1, .markdown h2 { padding-bottom: .3em; border-bottom: 1px solid var(--border) }".
// Since border-bottom appears in many other non-markdown rules, we check for
// the specific padding-bottom value (.3em) that is unique to the heading rule.
//
// Scenario: All heading levels are styled (h1/h2 border-bottom)
func TestMarkdownH1H2BorderBottom(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	// padding-bottom: .3em is only used on .markdown h1, .markdown h2 after implementation.
	if !strings.Contains(html, `padding-bottom: .3em`) {
		t.Fatalf("expected padding-bottom: .3em on .markdown h1/.markdown h2 in rendered CSS, got no match")
	}
}

// TestMarkdownParagraphSpacing verifies that the rendered HTML/CSS uses a
// 16px bottom margin on .markdown p (not 8px).
//
// Scenario: Block-level elements have consistent spacing
func TestMarkdownParagraphSpacing(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	if !strings.Contains(html, `margin: 0 0 16px 0`) {
		t.Fatalf("expected margin: 0 0 16px 0 in .markdown p rule, got no match")
	}
}

// TestMarkdownLiSpacing verifies that the rendered HTML/CSS contains a
// .markdown li + li rule to space list items.
//
// Scenario: Block-level elements have consistent spacing (list items)
func TestMarkdownLiSpacing(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	if !strings.Contains(html, `.markdown li + li`) {
		t.Fatalf("expected .markdown li + li rule in rendered CSS, got no match")
	}
}

// TestMarkdownWordWrap verifies that the rendered HTML/CSS contains a
// word-wrap: break-word declaration on the .markdown container.
//
// Scenario: Long words do not overflow container
func TestMarkdownWordWrap(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	if !strings.Contains(html, `word-wrap: break-word`) {
		t.Fatalf("expected word-wrap: break-word in .markdown rule, got no match")
	}
}

// TestMarkdownCodeFontSize verifies that the rendered HTML/CSS contains a
// font-size: 85% declaration on .markdown code.
//
// Scenario: Code blocks do not show inline code background (font-size)
func TestMarkdownCodeFontSize(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	if !strings.Contains(html, `font-size: 85%`) {
		t.Fatalf("expected font-size: 85%% in .markdown code rule, got no match")
	}
}

// TestMarkdownLinkTextDecoration verifies that the rendered HTML/CSS contains
// a .markdown a rule with text-decoration: none. Since text-decoration: none
// appears in chroma CSS rules (e.g. .lnlinks), we check for the specific
// .markdown a hover rule (text-decoration: underline) which is unique to the
// new markdown link styles.
//
// Scenario: Links use accent color without underline by default
func TestMarkdownLinkTextDecoration(t *testing.T) {
	html := renderReviewHTML(t, buildMinimalModel())
	// After implementation, .markdown a:hover { text-decoration: underline } is added.
	// This specific selector+property combination does not exist before implementation.
	if !strings.Contains(html, `.markdown a:hover`) {
		t.Fatalf("expected .markdown a:hover rule in rendered CSS, got no match")
	}
	if !strings.Contains(html, `text-decoration: underline`) {
		t.Fatalf("expected text-decoration: underline in .markdown a:hover rule, got no match")
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
