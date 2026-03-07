package app

import (
	"bytes"
	"html"
	"html/template"
	"net/url"
	"path/filepath"
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
	avatarBytes  = mustReadEmbeddedBytes("ai.png")
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

	// Render each top-level block.
	lastLine := fmLineCount
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		startByte, endByte := nodeByteRange(child)

		startLine := 0
		endLine := 0
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
		lastLine = endLine

		var buf bytes.Buffer
		if err := r.Render(&buf, source, child); err != nil {
			continue
		}
		blockHTML := rewriteMarkdownImageSources(buf.String(), baseDir)

		blocks = append(blocks, MarkdownBlock{
			StartLine: startLine,
			EndLine:   endLine,
			HTML:      blockHTML,
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
	root, err := xhtml.Parse(strings.NewReader(doc))
	if err != nil {
		return template.HTML(doc)
	}

	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "img" {
			for i := range n.Attr {
				if n.Attr[i].Key != "src" {
					continue
				}
				src := strings.TrimSpace(n.Attr[i].Val)
				if src == "" || isExternalAssetURL(src) {
					continue
				}
				rel := filepath.Clean(filepath.ToSlash(filepath.Join(baseDir, src)))
				n.Attr[i].Val = "/file?path=" + url.QueryEscape(rel)
			}
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
