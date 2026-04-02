package app

import (
	"strings"
	"testing"
)

func TestIsMarkdownPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "README.md", want: true},
		{path: "docs/guide.markdown", want: true},
		{path: "docs/Guide.MD", want: true},
		{path: "notes.mdx", want: false},
		{path: "main.go", want: false},
		{path: "README", want: false},
	}

	for _, tc := range tests {
		got := isMarkdownPath(tc.path)
		if got != tc.want {
			t.Fatalf("isMarkdownPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestUpdateFileViewMarkdownDefaultsToRendered(t *testing.T) {
	m := &ReviewModel{
		Files:        []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"# Heading", "", "Hello"}}},
		SelectedPath: "README.md",
		RenderFile:   true,
	}

	updateFileView(m)

	if !m.ViewFile.MarkdownFile {
		t.Fatal("expected markdown file flag")
	}
	if !m.ViewFile.MarkdownRendered {
		t.Fatal("expected markdown to render by default")
	}
	if len(m.ViewFile.MarkdownBlocks) == 0 {
		t.Fatal("expected markdown blocks")
	}
	foundH1 := false
	for _, b := range m.ViewFile.MarkdownBlocks {
		if strings.Contains(string(b.HTML), "<h1") {
			foundH1 = true
			break
		}
	}
	if !foundH1 {
		t.Fatal("expected at least one block containing <h1>")
	}
	if len(m.ViewFile.Lines) != 0 {
		t.Fatal("did not expect code lines in markdown rendered mode")
	}
}

func TestRenderMarkdownDocumentRewritesRelativeImagePaths(t *testing.T) {
	md := "![logo](internal/ui/logo.png)"
	html := string(renderMarkdownDocument("README.md", md))
	if !strings.Contains(html, `/file?path=internal%2Fui%2Flogo.png`) {
		t.Fatalf("expected rewritten local file URL, got %q", html)
	}
}

func TestRenderMarkdownDocumentKeepsExternalImagePaths(t *testing.T) {
	md := "![logo](https://example.com/logo.png)"
	html := string(renderMarkdownDocument("README.md", md))
	if !strings.Contains(html, `https://example.com/logo.png`) {
		t.Fatalf("expected external URL unchanged, got %q", html)
	}
}

func TestUpdateFileViewMarkdownCodeMode(t *testing.T) {
	m := &ReviewModel{
		Files:                []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"# Heading", "Hello"}}},
		SelectedPath:         "README.md",
		RenderFile:           true,
		MarkdownRenderByPath: map[string]bool{"README.md": false},
	}

	updateFileView(m)

	if !m.ViewFile.MarkdownFile {
		t.Fatal("expected markdown file flag")
	}
	if m.ViewFile.MarkdownRendered {
		t.Fatal("expected markdown code mode to stay selected")
	}
	if len(m.ViewFile.Lines) != 2 {
		t.Fatalf("expected 2 code lines, got %d", len(m.ViewFile.Lines))
	}
}

func TestRenderMarkdownBlocksLineNumbers(t *testing.T) {
	input := "# Heading\n\nParagraph text\nwith two lines.\n\n- item 1\n- item 2\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) != 4 {
		t.Fatalf("expected 4 blocks (heading, paragraph, list-item-1, list-item-2), got %d", len(blocks))
	}

	// Heading block should start at line 1.
	if blocks[0].StartLine != 1 {
		t.Errorf("heading block: StartLine = %d, want 1", blocks[0].StartLine)
	}
	if !strings.Contains(string(blocks[0].HTML), "<h1") {
		t.Errorf("heading block: expected <h1>, got %q", blocks[0].HTML)
	}

	// Paragraph block should start at line 3.
	if blocks[1].StartLine != 3 {
		t.Errorf("paragraph block: StartLine = %d, want 3", blocks[1].StartLine)
	}
	if !strings.Contains(string(blocks[1].HTML), "<p>") {
		t.Errorf("paragraph block: expected <p>, got %q", blocks[1].HTML)
	}

	// First list item block should start at line 6.
	if blocks[2].StartLine != 6 {
		t.Errorf("list item 1 block: StartLine = %d, want 6", blocks[2].StartLine)
	}
	if !strings.Contains(string(blocks[2].HTML), "item 1") {
		t.Errorf("list item 1 block: expected 'item 1', got %q", blocks[2].HTML)
	}
	if blocks[2].ListOpen == "" {
		t.Errorf("list item 1 block: expected non-empty ListOpen")
	}

	// Second list item block should start at line 7.
	if blocks[3].StartLine != 7 {
		t.Errorf("list item 2 block: StartLine = %d, want 7", blocks[3].StartLine)
	}
	if !strings.Contains(string(blocks[3].HTML), "item 2") {
		t.Errorf("list item 2 block: expected 'item 2', got %q", blocks[3].HTML)
	}
	if blocks[3].ListClose == "" {
		t.Errorf("list item 2 block: expected non-empty ListClose")
	}
}

func TestRenderMarkdownBlocksOrderedListWrapper(t *testing.T) {
	// Ordered list starting at 5 should use CSS counter-reset so that list
	// items rendered inside .md-block wrappers still number sequentially.
	input := "5. fifth\n6. sixth\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if !strings.Contains(string(blocks[0].ListOpen), "<ol") {
		t.Errorf("first block ListOpen: expected '<ol', got %q", blocks[0].ListOpen)
	}
	// counter-reset value is Start-1 (4) so the first increment gives 5.
	if !strings.Contains(string(blocks[0].ListOpen), "counter-reset: md-li-counter 4") {
		t.Errorf("first block ListOpen: expected counter-reset for start=5, got %q", blocks[0].ListOpen)
	}
	if !strings.Contains(string(blocks[1].ListClose), "</ol>") {
		t.Errorf("last block ListClose: expected '</ol>', got %q", blocks[1].ListClose)
	}

	// Ordered list starting at 1 should use counter-reset value 0.
	input2 := "1. first\n2. second\n"
	blocks2 := renderMarkdownBlocks("test.md", input2)

	if len(blocks2) < 1 {
		t.Fatalf("expected at least 1 block for 1-indexed ordered list, got %d", len(blocks2))
	}
	if !strings.Contains(string(blocks2[0].ListOpen), "counter-reset: md-li-counter 0") {
		t.Errorf("first block ListOpen for 1-indexed list: expected counter-reset 0, got %q", blocks2[0].ListOpen)
	}
}

func TestRenderMarkdownBlocksNestedList(t *testing.T) {
	// Nested list items belong to their parent item block; only top-level items are split.
	input := "- parent\n  - child1\n  - child2\n- sibling\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks (parent item and sibling item), got %d", len(blocks))
	}
	if !strings.Contains(string(blocks[0].HTML), "child1") {
		t.Errorf("parent block: expected nested 'child1' content, got %q", blocks[0].HTML)
	}
	if !strings.Contains(string(blocks[1].HTML), "sibling") {
		t.Errorf("sibling block: expected 'sibling' content, got %q", blocks[1].HTML)
	}
}

func TestRenderMarkdownBlocksTaskList(t *testing.T) {
	// GFM task list checkboxes should render as <input> elements.
	input := "- [ ] todo\n- [x] done\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if !strings.Contains(string(blocks[0].HTML), "<input") {
		t.Errorf("todo item block: expected '<input' checkbox element, got %q", blocks[0].HTML)
	}
	if !strings.Contains(string(blocks[1].HTML), "checked") {
		t.Errorf("done item block: expected 'checked' attribute, got %q", blocks[1].HTML)
	}
}

func TestRenderMarkdownBlocksNonListUnchanged(t *testing.T) {
	// Non-list blocks should have empty ListOpen and ListClose.
	input := "# Heading\n\nParagraph\n\n> Blockquote\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", len(blocks))
	}
	for i, b := range blocks {
		if b.ListOpen != "" {
			t.Errorf("block %d: expected empty ListOpen, got %q", i, b.ListOpen)
		}
		if b.ListClose != "" {
			t.Errorf("block %d: expected empty ListClose, got %q", i, b.ListClose)
		}
	}
}

func TestMarkdownBlocksListItemCommentProjection(t *testing.T) {
	m := &ReviewModel{
		Files:        []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"- item 1", "- item 2", "- item 3"}}},
		SelectedPath: "README.md",
		RenderFile:   true,
		Comments: []Comment{
			{ID: 1, Path: "README.md", StartLine: 2, EndLine: 2, Text: "Note on item 2"},
		},
		NextCommentID: 1,
	}

	updateFileView(m)

	blocks := m.ViewFile.MarkdownBlocks
	if len(blocks) != 3 {
		t.Fatalf("expected 3 list item blocks, got %d", len(blocks))
	}

	// Block at index 0 is item 1 (line 1) — should NOT be commented.
	if blocks[0].Commented {
		t.Errorf("item 1 block should not be commented")
	}

	// Block at index 1 is item 2 (line 2) — should be commented.
	if !blocks[1].Commented {
		t.Errorf("item 2 block should be commented")
	}
	if len(blocks[1].Comments) != 1 {
		t.Fatalf("item 2 block: expected 1 comment, got %d", len(blocks[1].Comments))
	}
	if blocks[1].Comments[0].Text != "Note on item 2" {
		t.Errorf("item 2 block comment text: got %q, want %q", blocks[1].Comments[0].Text, "Note on item 2")
	}

	// Block at index 2 is item 3 (line 3) — should NOT be commented.
	if blocks[2].Commented {
		t.Errorf("item 3 block should not be commented")
	}
}

func TestMarkdownBlocksListItemSelection(t *testing.T) {
	m := &ReviewModel{
		Files:          []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"- item 1", "- item 2", "- item 3"}}},
		SelectedPath:   "README.md",
		RenderFile:     true,
		SelectionStart: 2,
		SelectionEnd:   2,
	}

	updateFileView(m)

	blocks := m.ViewFile.MarkdownBlocks
	if len(blocks) != 3 {
		t.Fatalf("expected 3 list item blocks, got %d", len(blocks))
	}

	// Block at index 0 is item 1 (line 1) — should NOT be selected.
	if blocks[0].Selected {
		t.Errorf("item 1 block should not be selected")
	}

	// Block at index 1 is item 2 (line 2) — should be selected.
	if !blocks[1].Selected {
		t.Errorf("item 2 block should be selected")
	}

	// Block at index 2 is item 3 (line 3) — should NOT be selected.
	if blocks[2].Selected {
		t.Errorf("item 3 block should not be selected")
	}
}

func TestRenderMarkdownBlocksFrontmatterOffset(t *testing.T) {
	input := "---\ntitle: Test\n---\n# Heading\n\nBody\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) < 2 {
		t.Fatalf("expected at least 2 blocks (frontmatter + heading), got %d", len(blocks))
	}

	// Frontmatter block at lines 1-3.
	if blocks[0].StartLine != 1 {
		t.Errorf("frontmatter block: StartLine = %d, want 1", blocks[0].StartLine)
	}
	if blocks[0].EndLine != 3 {
		t.Errorf("frontmatter block: EndLine = %d, want 3", blocks[0].EndLine)
	}
	if !strings.Contains(string(blocks[0].HTML), "frontmatter") {
		t.Errorf("frontmatter block: expected frontmatter table, got %q", blocks[0].HTML)
	}

	// Heading block at line 4 (offset by 3 frontmatter lines).
	if blocks[1].StartLine != 4 {
		t.Errorf("heading block: StartLine = %d, want 4", blocks[1].StartLine)
	}
}

func TestMarkdownBlocksCommentsProjection(t *testing.T) {
	m := &ReviewModel{
		Files:        []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"# Heading", "", "Paragraph"}}},
		SelectedPath: "README.md",
		RenderFile:   true,
		Comments: []Comment{
			{ID: 1, Path: "README.md", StartLine: 3, EndLine: 3, Text: "Note on paragraph"},
		},
		NextCommentID: 1,
	}

	updateFileView(m)

	if len(m.ViewFile.MarkdownBlocks) < 2 {
		t.Fatalf("expected at least 2 blocks, got %d", len(m.ViewFile.MarkdownBlocks))
	}

	// The paragraph block should have the comment.
	paragraphBlock := m.ViewFile.MarkdownBlocks[1]
	if !paragraphBlock.Commented {
		t.Fatal("expected paragraph block to be marked as commented")
	}
	if len(paragraphBlock.Comments) != 1 {
		t.Fatalf("expected 1 comment on paragraph block, got %d", len(paragraphBlock.Comments))
	}
	if paragraphBlock.Comments[0].Text != "Note on paragraph" {
		t.Fatalf("unexpected comment text: %q", paragraphBlock.Comments[0].Text)
	}
}

func TestMarkdownBlocksSelection(t *testing.T) {
	m := &ReviewModel{
		Files:          []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"# Heading", "", "Paragraph"}}},
		SelectedPath:   "README.md",
		RenderFile:     true,
		SelectionStart: 3,
		SelectionEnd:   3,
	}

	updateFileView(m)

	if len(m.ViewFile.MarkdownBlocks) < 2 {
		t.Fatalf("expected at least 2 blocks, got %d", len(m.ViewFile.MarkdownBlocks))
	}

	// Heading block should NOT be selected.
	if m.ViewFile.MarkdownBlocks[0].Selected {
		t.Fatal("heading block should not be selected")
	}

	// Paragraph block should be selected (line 3 is in its range).
	if !m.ViewFile.MarkdownBlocks[1].Selected {
		t.Fatal("paragraph block should be selected")
	}
}

func TestUpdateFileViewMarkdownResetsToRenderedOnFileSwitch(t *testing.T) {
	m := &ReviewModel{
		Files:        []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"# Heading"}}},
		SelectedPath: "README.md",
		RenderFile:   true,
		ViewFile: ViewFile{
			Path:             "other.md",
			MarkdownRendered: false,
		},
	}

	updateFileView(m)

	if !m.ViewFile.MarkdownRendered {
		t.Fatal("expected markdown rendered mode after selecting a markdown file")
	}
}
