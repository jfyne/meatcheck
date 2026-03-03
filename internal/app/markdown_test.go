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

	if len(blocks) < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", len(blocks))
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

	// List block should start at line 6.
	if blocks[2].StartLine != 6 {
		t.Errorf("list block: StartLine = %d, want 6", blocks[2].StartLine)
	}
	if !strings.Contains(string(blocks[2].HTML), "<li>") {
		t.Errorf("list block: expected <li>, got %q", blocks[2].HTML)
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
