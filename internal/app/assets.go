package app

import (
	"bytes"
	"html"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/jfyne/meatcheck/internal/highlight"
	"github.com/jfyne/meatcheck/internal/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
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
