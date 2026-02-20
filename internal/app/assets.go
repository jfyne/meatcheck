package app

import (
	"bytes"
	"html"
	"html/template"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/jfyne/meatcheck/internal/highlight"
	"github.com/jfyne/meatcheck/internal/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
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
		goldmark.WithRendererOptions(),
	)
	codeRenderer = highlight.NewRenderer("github", "dracula", 4)
)

func renderMarkdown(input string) template.HTML {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(input), &buf); err != nil {
		return template.HTML(html.EscapeString(input))
	}
	return template.HTML(buf.String())
}

func renderMarkdownDocument(path string, input string) template.HTML {
	baseDir := filepath.Dir(path)
	if baseDir == "." {
		baseDir = ""
	}
	rendered := renderMarkdown(input)
	return rewriteMarkdownImageSources(string(rendered), baseDir)
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
