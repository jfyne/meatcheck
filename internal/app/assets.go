package app

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jfyne/meatcheck/internal/highlight"
	"github.com/jfyne/meatcheck/internal/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	xhtml "golang.org/x/net/html"
)

var (
	templateHTML = mustReadEmbedded("template.html")
	stylesCSS    = mustReadEmbedded("styles.css")
	logoBytes    = mustReadEmbeddedBytes("logo.png")
)

var (
	markdownRenderer = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
	)
	codeRenderer = highlight.NewRenderer("github", "dracula", 4)
)

// renderFrontmatter extracts YAML frontmatter from the start of input (if present),
// renders it as an HTML table, and returns the table HTML along with the remaining content.
// Frontmatter is detected when input starts with "---\n" and has a closing "\n---\n" or "\n---" at EOF.
// If no valid frontmatter is found, fmHTML is empty and rest equals input.
func renderFrontmatter(input string) (fmHTML string, rest string) {
	const opener = "---\n"
	if !strings.HasPrefix(input, opener) {
		return "", input
	}

	// Search for closing delimiter after the opening "---\n"
	body := input[len(opener):]
	closeIdx := -1
	afterClose := ""

	const closerMid = "\n---\n"
	if idx := strings.Index(body, closerMid); idx >= 0 {
		closeIdx = idx
		afterClose = body[idx+len(closerMid):]
	} else {
		const closerEOF = "\n---"
		if strings.HasSuffix(body, closerEOF) {
			closeIdx = len(body) - len(closerEOF)
			afterClose = ""
		}
	}

	if closeIdx < 0 {
		return "", input
	}

	yamlBlock := body[:closeIdx]

	var rows strings.Builder
	for line := range strings.SplitSeq(yamlBlock, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		rows.WriteString("<tr><td class=\"frontmatter-key\">")
		rows.WriteString(html.EscapeString(key))
		rows.WriteString("</td><td>")
		rows.WriteString(html.EscapeString(value))
		rows.WriteString("</td></tr>\n")
	}

	if rows.Len() == 0 {
		return "", afterClose
	}
	fmHTML = "<table class=\"frontmatter\">\n<tbody>\n" + rows.String() + "</tbody>\n</table>\n"
	return fmHTML, afterClose
}

func renderMarkdown(input string) template.HTML {
	fmHTML, rest := renderFrontmatter(input)
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(rest), &buf); err != nil {
		return template.HTML(fmHTML + html.EscapeString(rest))
	}
	return template.HTML(fmHTML + buf.String())
}

func renderMarkdownDocument(path string, input string) template.HTML {
	baseDir := filepath.Dir(path)
	if baseDir == "." {
		baseDir = ""
	}
	rendered := renderMarkdown(input)
	return rewriteMarkdownImageSources(string(rendered), baseDir)
}

// resolveLineRange converts a node's byte range into 1-based start/end line
// numbers. If the node has no byte range, startLine defaults to lastLine+1.
// endLine is always >= startLine.
func resolveLineRange(node ast.Node, lastLine int, byteToLine func(int) int) (startLine, endLine int) {
	startByte, endByte := nodeByteRange(node)
	if startByte >= 0 {
		startLine = byteToLine(startByte)
	} else {
		startLine = lastLine + 1
	}
	if endByte > 0 {
		endLine = byteToLine(endByte - 1)
	}
	if endLine < startLine {
		endLine = startLine
	}
	return startLine, endLine
}

// renderMarkdownBlocks parses markdown into per-block HTML chunks with source line mappings.
func renderMarkdownBlocks(path, input string) []MarkdownBlock {
	baseDir := filepath.Dir(path)
	if baseDir == "." {
		baseDir = ""
	}

	fmHTML, rest := renderFrontmatter(input)
	fmLineCount := 0
	if fmHTML != "" {
		prefix := input[:len(input)-len(rest)]
		fmLineCount = strings.Count(prefix, "\n")
		if len(prefix) > 0 && !strings.HasSuffix(prefix, "\n") {
			fmLineCount++
		}
	}

	source := []byte(rest)

	// Build byte-offset to line-number lookup (sorted line start offsets).
	lineStarts := []int{0}
	for i, b := range source {
		if b == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	byteToLine := func(offset int) int {
		idx := max(sort.SearchInts(lineStarts, offset+1)-1, 0)
		return idx + 1 + fmLineCount // 1-based, offset by frontmatter
	}

	// Parse AST.
	reader := text.NewReader(source)
	doc := markdownRenderer.Parser().Parse(reader)
	r := markdownRenderer.Renderer()

	var blocks []MarkdownBlock

	// Add frontmatter block if present.
	if fmHTML != "" {
		blocks = append(blocks, MarkdownBlock{
			StartLine: 1,
			EndLine:   fmLineCount,
			HTML:      template.HTML(fmHTML),
		})
	}

	// Render each top-level block. A single buffer is reused across iterations.
	var buf bytes.Buffer
	lastLine := fmLineCount
	// Folds opened by raw <details> markup and not yet closed, outermost first.
	var foldStack []int
	lastFoldID := 0
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if listNode, ok := child.(*ast.List); ok {
			// Build wrapper tags.
			var openTag, closeTag string
			if listNode.IsOrdered() {
				// Use a CSS counter-reset inline style so that <li> elements
				// wrapped in .md-block divs (non-direct children of <ol>)
				// still display sequential numbers.
				openTag = fmt.Sprintf(`<ol class="md-list" style="counter-reset: md-li-counter %d">`, listNode.Start-1)
				closeTag = "</ol>"
			} else {
				openTag = `<ul class="md-list">`
				closeTag = "</ul>"
			}

			// Collect top-level list items.
			var items []ast.Node
			for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
				items = append(items, item)
			}

			for i, item := range items {
				startLine, endLine := resolveLineRange(item, lastLine, byteToLine)
				lastLine = endLine

				buf.Reset()
				if err := r.Render(&buf, source, item); err != nil {
					continue
				}
				blockHTML := rewriteMarkdownImageSources(buf.String(), baseDir)

				block := MarkdownBlock{
					StartLine: startLine,
					EndLine:   endLine,
					HTML:      blockHTML,
					InFolds:   slices.Clone(foldStack),
				}
				if i == 0 {
					block.ListOpen = template.HTML(openTag)
				}
				if i == len(items)-1 {
					block.ListClose = template.HTML(closeTag)
				}
				blocks = append(blocks, block)
			}
			continue
		}

		startLine, endLine := resolveLineRange(child, lastLine, byteToLine)
		lastLine = endLine

		buf.Reset()
		if err := r.Render(&buf, source, child); err != nil {
			continue
		}
		rendered := buf.String()

		// Raw HTML that leaves an element open — a <details> fold, a <div>
		// wrapper — has to render outside the block divs, otherwise the div
		// closes the element before the blocks it was meant to enclose.
		if child.Kind() == ast.KindHTMLBlock {
			if shape := inspectRawHTML(rendered); !shape.balanced {
				raw := rewriteRawHTMLImageSources(rendered, baseDir)
				if raw != rendered {
					shape = inspectRawHTML(raw)
				}

				block := MarkdownBlock{
					StartLine: startLine,
					EndLine:   endLine,
					HTML:      template.HTML(raw),
					HTMLOpen:  template.HTML(shape.openHTML),
					Wrapper:   true,
				}
				for range shape.closesDetails {
					if n := len(foldStack); n > 0 {
						foldStack = foldStack[:n-1]
					}
				}
				for range shape.opensDetails {
					lastFoldID++
					foldStack = append(foldStack, lastFoldID)
					block.OpensFolds = append(block.OpensFolds, lastFoldID)
				}
				blocks = append(blocks, block)
				continue
			}
		}

		blocks = append(blocks, MarkdownBlock{
			StartLine: startLine,
			EndLine:   endLine,
			HTML:      rewriteMarkdownImageSources(rendered, baseDir),
			InFolds:   slices.Clone(foldStack),
		})
	}

	return blocks
}

// nodeByteRange returns the [start, end) byte range of an AST node's source text,
// searching the node's own lines and recursing into children.
func nodeByteRange(node ast.Node) (start, end int) {
	start, end = -1, -1

	// Lines() is only valid for block nodes; calling it on inline nodes panics.
	if node.Type() == ast.TypeBlock {
		if segs := node.Lines(); segs != nil && segs.Len() > 0 {
			s := segs.At(0).Start
			e := segs.At(segs.Len() - 1).Stop
			if start < 0 || s < start {
				start = s
			}
			if end < 0 || e > end {
				end = e
			}
		}
	}

	if t, ok := node.(*ast.Text); ok {
		s := t.Segment.Start
		e := t.Segment.Stop
		if s != e {
			if start < 0 || s < start {
				start = s
			}
			if end < 0 || e > end {
				end = e
			}
		}
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		cs, ce := nodeByteRange(child)
		if cs >= 0 && (start < 0 || cs < start) {
			start = cs
		}
		if ce >= 0 && (end < 0 || ce > end) {
			end = ce
		}
	}

	return start, end
}

func rewriteMarkdownImageSources(doc string, baseDir string) template.HTML {
	// Fast-path: skip the full HTML parse when there are no images to rewrite.
	if !strings.Contains(doc, "<img") {
		return template.HTML(doc)
	}

	root, err := xhtml.Parse(strings.NewReader(doc))
	if err != nil {
		return template.HTML(doc)
	}

	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "img" {
			rewriteImageAttrs(n.Attr, baseDir)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)

	var out bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := xhtml.Render(&out, c); err != nil {
			return template.HTML(doc)
		}
	}
	return template.HTML(out.String())
}

// rewriteRawHTMLImageSources rewrites local image sources token by token. A
// wrapper block is unbalanced on purpose, so it cannot go through the document
// parse in rewriteMarkdownImageSources, which would close its open tags.
func rewriteRawHTMLImageSources(raw string, baseDir string) string {
	if !strings.Contains(raw, "<img") {
		return raw
	}

	var out strings.Builder
	z := xhtml.NewTokenizer(strings.NewReader(raw))
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			break
		}
		// Raw's contents are only valid until the next tokenizer call.
		text := string(z.Raw())
		if tt != xhtml.StartTagToken && tt != xhtml.SelfClosingTagToken {
			out.WriteString(text)
			continue
		}
		token := z.Token()
		if token.Data != "img" {
			out.WriteString(text)
			continue
		}
		rewriteImageAttrs(token.Attr, baseDir)
		out.WriteString(token.String())
	}
	return out.String()
}

// rewriteImageAttrs points a local image source at the file endpoint, in place.
func rewriteImageAttrs(attrs []xhtml.Attribute, baseDir string) {
	for i := range attrs {
		if attrs[i].Key != "src" {
			continue
		}
		src := strings.TrimSpace(attrs[i].Val)
		if src == "" || isExternalAssetURL(src) {
			continue
		}
		rel := filepath.Clean(filepath.ToSlash(filepath.Join(baseDir, src)))
		attrs[i].Val = "/file?path=" + url.QueryEscape(rel)
	}
}

// voidHTMLElements never carry a closing tag, so they leave a raw HTML block
// balanced.
var voidHTMLElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// rawHTMLShape describes how a raw HTML block nests.
type rawHTMLShape struct {
	// balanced reports whether the block closes every element it opens and
	// opens every element it closes. An unbalanced block encloses the blocks
	// around it rather than standing on its own.
	balanced bool
	// opensDetails counts the <details> elements the block leaves open;
	// closesDetails counts the </details> tags closing an earlier one.
	opensDetails  int
	closesDetails int
	// openHTML is the block's HTML with an open attribute on each <details> it
	// leaves open. Empty when it leaves none open.
	openHTML string
}

// inspectRawHTML walks the tags of a raw HTML block, tracking which elements it
// leaves open and which it closes without having opened.
func inspectRawHTML(raw string) rawHTMLShape {
	shape := rawHTMLShape{balanced: true}

	type openTag struct {
		name        string
		tagEnd      int // offset of the '>' closing the start tag
		alreadyOpen bool
	}
	var stack []openTag

	z := xhtml.NewTokenizer(strings.NewReader(raw))
	offset := 0
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			break
		}
		// Read the length before TagName, which may rewrite Raw's buffer.
		offset += len(z.Raw())

		switch tt {
		case xhtml.StartTagToken:
			name, hasAttr := z.TagName()
			tag := string(name)
			if voidHTMLElements[tag] {
				continue
			}
			open := openTag{name: tag, tagEnd: offset - 1}
			for hasAttr {
				var key []byte
				key, _, hasAttr = z.TagAttr()
				if string(key) == "open" {
					open.alreadyOpen = true
				}
			}
			stack = append(stack, open)
		case xhtml.EndTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if n := len(stack); n > 0 && stack[n-1].name == tag {
				stack = stack[:n-1]
				continue
			}
			shape.balanced = false
			if tag == "details" {
				shape.closesDetails++
			}
		}
	}

	if len(stack) > 0 {
		shape.balanced = false
	}

	var forceOpen []int
	for _, tag := range stack {
		if tag.name != "details" {
			continue
		}
		shape.opensDetails++
		// Splice only into a start tag the tokenizer read to its '>'.
		if !tag.alreadyOpen && tag.tagEnd < len(raw) && raw[tag.tagEnd] == '>' {
			forceOpen = append(forceOpen, tag.tagEnd)
		}
	}
	if shape.opensDetails > 0 {
		shape.openHTML = insertAtOffsets(raw, forceOpen, " open")
	}

	return shape
}

// insertAtOffsets splices text into raw at each offset, ascending.
func insertAtOffsets(raw string, offsets []int, text string) string {
	if len(offsets) == 0 {
		return raw
	}
	var out strings.Builder
	prev := 0
	for _, offset := range offsets {
		out.WriteString(raw[prev:offset])
		out.WriteString(text)
		prev = offset
	}
	out.WriteString(raw[prev:])
	return out.String()
}

func isExternalAssetURL(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "#") ||
		strings.HasPrefix(lower, "/")
}

func isMarkdownPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

func buildCSS() string {
	var buf bytes.Buffer
	buf.WriteString(stylesCSS)
	buf.WriteString("\n")
	buf.WriteString(codeRenderer.BuildCSS())
	return buf.String()
}

func mustReadEmbedded(path string) string {
	data, err := ui.FS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func mustReadEmbeddedBytes(path string) []byte {
	data, err := ui.FS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}
