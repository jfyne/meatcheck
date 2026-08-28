package app

import (
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
)

// foldSource is a markdown document with a <details> fold whose content is
// separated by blank lines, which is how goldmark splits the opening tag, the
// content and the closing tag into separate top-level blocks.
var foldSource = strings.Join([]string{
	"# Title",
	"",
	"<details>",
	"<summary>Show the detail</summary>",
	"",
	"Hidden paragraph",
	"",
	"</details>",
	"",
	"After",
}, "\n")

func foldModel() *ReviewModel {
	return &ReviewModel{
		Files:        []File{{Path: "README.md", PathSlash: "README.md", Lines: strings.Split(foldSource, "\n")}},
		SelectedPath: "README.md",
		RenderFile:   true,
	}
}

// TestHTTPRenderMarkdownFoldHidesItsContent renders the whole page and reads it
// back with an HTML parser, which is what the browser does: the fold only hides
// anything if the content blocks are children of the <details> element.
func TestHTTPRenderMarkdownFoldHidesItsContent(t *testing.T) {
	m := foldModel()
	m.Mode = ModeFile
	m.RenderComments = true
	m.Viewed = make(map[string]bool)
	m.Ranges = map[string][]LineRange{}
	m.MarkdownRenderByPath = map[string]bool{}
	m.Tree = buildTree(m.Files, m.SelectedPath, nil, nil)

	doc, err := xhtml.Parse(strings.NewReader(renderReviewHTML(t, m)))
	if err != nil {
		t.Fatalf("parse rendered page: %v", err)
	}

	details := findElement(doc, "details")
	if details == nil {
		t.Fatal("expected a <details> element in the rendered page")
	}
	if hasAttribute(details, "open") {
		t.Error("expected the fold to render collapsed by default")
	}

	folded := elementText(details)
	if !strings.Contains(folded, "Hidden paragraph") {
		t.Errorf("expected the fold to enclose its content, but the browser sees %q inside it", folded)
	}
	if strings.Contains(folded, "After") {
		t.Errorf("expected the fold to end before the following block, but the browser sees %q inside it", folded)
	}
	if findClassedElement(details, "md-block") == nil {
		t.Error("expected a commentable .md-block inside the fold")
	}
}

func findElement(n *xhtml.Node, tag string) *xhtml.Node {
	if n.Type == xhtml.ElementNode && n.Data == tag {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func findClassedElement(n *xhtml.Node, class string) *xhtml.Node {
	if n.Type == xhtml.ElementNode {
		for _, c := range strings.Fields(attrValue(n, "class")) {
			if c == class {
				return n
			}
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findClassedElement(child, class); found != nil {
			return found
		}
	}
	return nil
}

func hasAttribute(n *xhtml.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func attrValue(n *xhtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func elementText(n *xhtml.Node) string {
	var sb strings.Builder
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			sb.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return sb.String()
}

func TestRenderMarkdownBlocksMarksUnbalancedHTMLAsWrapper(t *testing.T) {
	blocks := renderMarkdownBlocks("test.md", foldSource)

	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks (title, fold open, paragraph, fold close, after), got %d", len(blocks))
	}

	open := blocks[1]
	if !open.Wrapper {
		t.Errorf("fold open block: expected Wrapper, got %+v", open)
	}
	if !strings.Contains(string(open.HTML), "<details>") {
		t.Errorf("fold open block: expected <details>, got %q", open.HTML)
	}
	if len(open.OpensFolds) != 1 {
		t.Errorf("fold open block: expected 1 opened fold, got %v", open.OpensFolds)
	}
	if !strings.Contains(string(open.HTMLOpen), "<details open>") {
		t.Errorf("fold open block: expected an open variant, got %q", open.HTMLOpen)
	}

	inner := blocks[2]
	if inner.Wrapper {
		t.Errorf("paragraph block: expected a commentable block, got a wrapper")
	}
	if !strings.Contains(string(inner.HTML), "Hidden paragraph") {
		t.Errorf("paragraph block: expected the fold content, got %q", inner.HTML)
	}
	if len(open.OpensFolds) == 1 && (len(inner.InFolds) != 1 || inner.InFolds[0] != open.OpensFolds[0]) {
		t.Errorf("paragraph block: expected to sit inside fold %v, got %v", open.OpensFolds, inner.InFolds)
	}

	closer := blocks[3]
	if !closer.Wrapper {
		t.Errorf("fold close block: expected Wrapper, got %+v", closer)
	}
	if !strings.Contains(string(closer.HTML), "</details>") {
		t.Errorf("fold close block: expected </details>, got %q", closer.HTML)
	}

	if after := blocks[4]; len(after.InFolds) != 0 {
		t.Errorf("trailing block: expected to sit outside the fold, got %v", after.InFolds)
	}
}

func TestRenderMarkdownBlocksKeepsBalancedHTMLCommentable(t *testing.T) {
	// A self-contained raw HTML block closes everything it opens, so it stays
	// a normal block the reviewer can select and comment on.
	input := "<details><summary>All in one</summary>Body</details>\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Wrapper {
		t.Error("expected a balanced HTML block to stay commentable, got a wrapper")
	}
}

func TestRenderMarkdownBlocksWrapsNonDetailsHTML(t *testing.T) {
	// The same breakage applies to any raw HTML element spanning blocks.
	input := "<div align=\"center\">\n\nCentred paragraph\n\n</div>\n"
	blocks := renderMarkdownBlocks("test.md", input)

	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks (div open, paragraph, div close), got %d", len(blocks))
	}
	if !blocks[0].Wrapper || !blocks[2].Wrapper {
		t.Errorf("expected the div tags to be wrappers, got %+v and %+v", blocks[0], blocks[2])
	}
	if blocks[0].HTMLOpen != "" {
		t.Errorf("expected no forced-open variant for a non-details wrapper, got %q", blocks[0].HTMLOpen)
	}
	if len(blocks[1].InFolds) != 0 {
		t.Errorf("expected a <div> wrapper not to register as a fold, got %v", blocks[1].InFolds)
	}
}

func TestMarkdownFoldOpensWhenItHoldsAComment(t *testing.T) {
	m := foldModel()
	m.Comments = []Comment{
		{ID: 1, Path: "README.md", StartLine: 6, EndLine: 6, Text: "Note on the hidden paragraph"},
	}
	m.NextCommentID = 1

	updateFileView(m)

	blocks := m.ViewFile.MarkdownBlocks
	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(blocks))
	}
	if !blocks[2].Commented {
		t.Fatal("expected the paragraph inside the fold to carry the comment")
	}
	if !strings.Contains(string(blocks[1].HTML), "<details open>") {
		t.Fatalf("expected the fold holding a comment to render open, got %q", blocks[1].HTML)
	}
}

func TestMarkdownFoldOpensWhenItHoldsTheSelection(t *testing.T) {
	m := foldModel()
	m.SelectionStart = 6
	m.SelectionEnd = 6

	updateFileView(m)

	blocks := m.ViewFile.MarkdownBlocks
	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(blocks))
	}
	if !strings.Contains(string(blocks[1].HTML), "<details open>") {
		t.Fatalf("expected the fold holding the selection to render open, got %q", blocks[1].HTML)
	}
}

func TestMarkdownWrapperCommentsSurfaceOnTheNextBlock(t *testing.T) {
	// Wrapper markup renders outside the block divs, so a comment left on the
	// <details> line in the code view has to surface on the block that follows
	// it rather than disappearing from the rendered view.
	m := foldModel()
	m.Comments = []Comment{
		{ID: 1, Path: "README.md", StartLine: 3, EndLine: 4, Text: "Should this fold be open?"},
	}
	m.NextCommentID = 1

	updateFileView(m)

	blocks := m.ViewFile.MarkdownBlocks
	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(blocks))
	}
	inner := blocks[2]
	if len(inner.Comments) != 1 {
		t.Fatalf("expected the fold's comment to surface on the first block inside it, got %d comments", len(inner.Comments))
	}
	if inner.Comments[0].Text != "Should this fold be open?" {
		t.Fatalf("unexpected comment text: %q", inner.Comments[0].Text)
	}
	if !strings.Contains(string(blocks[1].HTML), "<details open>") {
		t.Fatalf("expected the fold to open around its own comment, got %q", blocks[1].HTML)
	}
	if blocks[2].StartLine != 6 {
		t.Errorf("expected the block to keep its own anchor line 6, got %d", blocks[2].StartLine)
	}
}

func TestMarkdownTrailingWrapperCommentsSurfaceOnTheLastBlock(t *testing.T) {
	// A fold that closes at the end of the document has no block after it, so
	// its comments belong to the block before it.
	input := "Intro\n\n<details>\n<summary>Last</summary>\n\nInside the fold\n\n</details>\n"
	m := &ReviewModel{
		Files:        []File{{Path: "README.md", PathSlash: "README.md", Lines: strings.Split(strings.TrimSuffix(input, "\n"), "\n")}},
		SelectedPath: "README.md",
		RenderFile:   true,
		Comments: []Comment{
			{ID: 1, Path: "README.md", StartLine: 8, EndLine: 8, Text: "Comment on the closing tag"},
		},
		NextCommentID: 1,
	}

	updateFileView(m)

	blocks := m.ViewFile.MarkdownBlocks
	inner := blocks[len(blocks)-2]
	if inner.Wrapper {
		t.Fatalf("expected the last content block before the closing wrapper, got a wrapper")
	}
	if len(inner.Comments) != 1 {
		t.Fatalf("expected the closing tag's comment to surface on the block before it, got %d comments", len(inner.Comments))
	}
}

func TestMarkdownFoldStaysClosedWithoutComments(t *testing.T) {
	m := foldModel()

	updateFileView(m)

	blocks := m.ViewFile.MarkdownBlocks
	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(blocks))
	}
	if strings.Contains(string(blocks[1].HTML), "open") {
		t.Fatalf("expected the fold to render collapsed, got %q", blocks[1].HTML)
	}
}
