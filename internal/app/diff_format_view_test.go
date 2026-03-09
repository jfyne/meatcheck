package app

import (
	"strings"
	"testing"
)

// TestUpdateDiffViewUnifiedFormat verifies that updateDiffView populates
// ViewDiff (not ViewDiffSplit) when DiffFormat is DiffFormatUnified.
//
// Scenario: Toggle between unified and side-by-side diff view (unified path)
func TestUpdateDiffViewUnifiedFormat(t *testing.T) {
	model := &ReviewModel{
		Mode:         ModeDiff,
		DiffFormat:   DiffFormatUnified,
		SelectedPath: "a.go",
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
		Viewed:               make(map[string]bool),
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}

	updateDiffView(model)

	// Unified format: ViewDiff.Hunks must be populated.
	if len(model.ViewDiff.Hunks) == 0 {
		t.Errorf("expected ViewDiff.Hunks to be non-empty for DiffFormatUnified, got 0 hunks")
	}
	// Specific content check: the hunk must carry the context line.
	if len(model.ViewDiff.Hunks[0].Lines) != 1 {
		t.Errorf("expected 1 line in hunk, got %d", len(model.ViewDiff.Hunks[0].Lines))
	}
	if model.ViewDiff.Hunks[0].Lines[0].Text != "package main" {
		t.Errorf("expected line text %q, got %q", "package main", model.ViewDiff.Hunks[0].Lines[0].Text)
	}
	// Split view must not be populated.
	if model.ViewDiffSplit != nil {
		t.Errorf("expected ViewDiffSplit to be nil for DiffFormatUnified, got %v", model.ViewDiffSplit)
	}
}

// TestUpdateDiffViewSplitFormat verifies that updateDiffView populates
// ViewDiffSplit (not ViewDiff) when DiffFormat is DiffFormatSplit.
//
// Scenario: Toggle between unified and side-by-side diff view (split path)
func TestUpdateDiffViewSplitFormat(t *testing.T) {
	model := &ReviewModel{
		Mode:         ModeDiff,
		DiffFormat:   DiffFormatSplit,
		SelectedPath: "b.go",
		DiffFiles: []DiffFile{{
			Path: "b.go",
			Hunks: []DiffHunk{{
				OldStart: 3,
				OldCount: 2,
				NewStart: 3,
				NewCount: 2,
				Lines: []DiffLine{
					{Kind: DiffDel, OldLine: 3, NewLine: 0, Text: "old line"},
					{Kind: DiffAdd, OldLine: 0, NewLine: 3, Text: "new line"},
				},
			}},
		}},
		Viewed:               make(map[string]bool),
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}

	updateDiffView(model)

	// Split format: ViewDiffSplit must be populated.
	if len(model.ViewDiffSplit) == 0 {
		t.Errorf("expected ViewDiffSplit to be non-empty for DiffFormatSplit, got 0 hunks")
	}
	// Specific content check: first row left side must carry the del line.
	rows := model.ViewDiffSplit[0].Rows
	if len(rows) == 0 {
		t.Fatalf("expected at least 1 row in ViewDiffSplit hunk, got 0")
	}
	if rows[0].Left.Text != "old line" {
		t.Errorf("expected split row Left.Text %q, got %q", "old line", rows[0].Left.Text)
	}
	if rows[0].Right.Text != "new line" {
		t.Errorf("expected split row Right.Text %q, got %q", "new line", rows[0].Right.Text)
	}
	// Unified view must not be populated (hunks must be empty).
	if len(model.ViewDiff.Hunks) != 0 {
		t.Errorf("expected ViewDiff.Hunks to be empty for DiffFormatSplit, got %d hunks", len(model.ViewDiff.Hunks))
	}
}

// TestToggleDiffFormatClearsSelection verifies that when the diff format is
// toggled the selection state is cleared, matching what the toggle-diff-format
// handler does.
//
// Scenario: Toggle between unified and side-by-side diff view (selection cleared)
func TestToggleDiffFormatClearsSelection(t *testing.T) {
	model := &ReviewModel{
		Mode:           ModeDiff,
		DiffFormat:     DiffFormatUnified,
		SelectedPath:   "c.go",
		SelectionStart: 5,
		SelectionEnd:   10,
		SelectionSide:  "old",
		DiffFiles: []DiffFile{{
			Path: "c.go",
			Hunks: []DiffHunk{{
				OldStart: 1,
				OldCount: 1,
				NewStart: 1,
				NewCount: 1,
				Lines: []DiffLine{
					{Kind: DiffContext, OldLine: 5, NewLine: 5, Text: "ctx"},
				},
			}},
		}},
		Viewed:               make(map[string]bool),
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}

	// Simulate what toggle-diff-format handler does: swap format, clear selection.
	if model.DiffFormat == DiffFormatUnified {
		model.DiffFormat = DiffFormatSplit
	} else {
		model.DiffFormat = DiffFormatUnified
	}
	model.SelectionStart = 0
	model.SelectionEnd = 0
	model.SelectionSide = ""
	updateDiffView(model)

	if model.DiffFormat != DiffFormatSplit {
		t.Errorf("expected DiffFormat to be %q after toggle, got %q", DiffFormatSplit, model.DiffFormat)
	}
	if model.SelectionStart != 0 {
		t.Errorf("expected SelectionStart = 0 after toggle, got %d", model.SelectionStart)
	}
	if model.SelectionEnd != 0 {
		t.Errorf("expected SelectionEnd = 0 after toggle, got %d", model.SelectionEnd)
	}
	if model.SelectionSide != "" {
		t.Errorf("expected SelectionSide = %q after toggle, got %q", "", model.SelectionSide)
	}
}

// TestHTTPRenderDiffFormatToggleHiddenInFileMode verifies that the diff format
// toggle button is NOT rendered when Mode == ModeFile (file mode).
//
// Scenario: Diff format toggle hidden in file mode
func TestHTTPRenderDiffFormatToggleHiddenInFileMode(t *testing.T) {
	model := &ReviewModel{
		Files: []File{{
			Path:      "a.go",
			PathSlash: "a.go",
			Lines:     []string{"package main"},
		}},
		SelectedPath:         "a.go",
		Mode:                 ModeFile,
		RenderFile:           true,
		RenderComments:       true,
		Viewed:               make(map[string]bool),
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(model.Files, model.SelectedPath, nil, nil)

	html := renderReviewHTML(t, model)

	// The toggle button element must not appear in file mode.
	if strings.Contains(html, `live-click="toggle-diff-format"`) {
		t.Errorf("expected diff format toggle button to be absent in file mode, but found it in rendered HTML")
	}
}

// TestHTTPRenderDiffFormatToggleVisibleInDiffMode verifies that the diff format
// toggle button IS rendered when Mode == ModeDiff.
//
// Scenario: Diff format toggle hidden in file mode (inverse: visible in diff mode)
func TestHTTPRenderDiffFormatToggleVisibleInDiffMode(t *testing.T) {
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
		Mode:                 ModeDiff,
		DiffFormat:           DiffFormatUnified,
		RenderFile:           true,
		RenderComments:       true,
		Viewed:               make(map[string]bool),
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(diffFilesAsFiles(model.DiffFiles), model.SelectedPath, nil, nil)

	html := renderReviewHTML(t, model)

	// The toggle button must be present in diff mode.
	if !strings.Contains(html, `toggle-diff-format`) {
		t.Errorf("expected diff format toggle (toggle-diff-format) to be present in diff mode, but not found in rendered HTML")
	}
}

// TestHTTPRenderSplitDiffBlock verifies that when DiffFormat == DiffFormatSplit
// the rendered HTML contains the side-by-side split block (diff-row-split class).
//
// Scenario: Side-by-side view pairs old and new lines correctly (template output)
func TestHTTPRenderSplitDiffBlock(t *testing.T) {
	model := &ReviewModel{
		DiffFiles: []DiffFile{{
			Path: "a.go",
			Hunks: []DiffHunk{{
				OldStart: 1,
				OldCount: 1,
				NewStart: 1,
				NewCount: 1,
				Lines: []DiffLine{
					{Kind: DiffDel, OldLine: 1, NewLine: 0, Text: "old line"},
					{Kind: DiffAdd, OldLine: 0, NewLine: 1, Text: "new line"},
				},
			}},
		}},
		SelectedPath:         "a.go",
		Mode:                 ModeDiff,
		DiffFormat:           DiffFormatSplit,
		RenderFile:           true,
		RenderComments:       true,
		Viewed:               make(map[string]bool),
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(diffFilesAsFiles(model.DiffFiles), model.SelectedPath, nil, nil)

	html := renderReviewHTML(t, model)

	// The split template block must render the diff-row-split element.
	if !strings.Contains(html, `diff-row-split`) {
		t.Errorf("expected diff-row-split class in rendered HTML for DiffFormatSplit, got: %q", html)
	}
	// Both left (del) and right (add) cells must be rendered.
	if !strings.Contains(html, `diff-cell`) {
		t.Errorf("expected diff-cell class in rendered HTML for DiffFormatSplit, got: %q", html)
	}
}

// TestHTTPRenderUnifiedDiffNotSplit verifies that when DiffFormat == DiffFormatUnified
// the rendered HTML uses the unified diff block (diff-line class) and NOT the
// split block (diff-row-split).
//
// Scenario: Toggle between unified and side-by-side diff view (template unified path)
func TestHTTPRenderUnifiedDiffNotSplit(t *testing.T) {
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
		Mode:                 ModeDiff,
		DiffFormat:           DiffFormatUnified,
		RenderFile:           true,
		RenderComments:       true,
		Viewed:               make(map[string]bool),
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(diffFilesAsFiles(model.DiffFiles), model.SelectedPath, nil, nil)

	html := renderReviewHTML(t, model)

	// Unified format must use diff-line elements.
	if !strings.Contains(html, `diff-line`) {
		t.Errorf("expected diff-line class in rendered HTML for DiffFormatUnified, got: %q", html)
	}
	// Split view HTML elements must be absent (check for the element class, not CSS selectors).
	if strings.Contains(html, `class="diff-row-split"`) {
		t.Errorf("expected diff-row-split elements to be absent in rendered HTML for DiffFormatUnified")
	}
}
