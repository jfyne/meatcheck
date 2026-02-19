package highlight

import (
	"bytes"
	"html"
	"html/template"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

type Renderer struct {
	formatter *chromahtml.Formatter
	light     *chroma.Style
	dark      *chroma.Style
}

func NewRenderer(lightStyle, darkStyle string, tabWidth int) *Renderer {
	formatter := chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.TabWidth(tabWidth),
	)
	light := styles.Get(lightStyle)
	if light == nil {
		light = styles.Fallback
	}
	dark := styles.Get(darkStyle)
	if dark == nil {
		dark = styles.Fallback
	}
	return &Renderer{
		formatter: formatter,
		light:     light,
		dark:      dark,
	}
}

func (r *Renderer) RenderLines(path string, lines []string) []template.HTML {
	lexer := resolveLexer(path, lines)
	if lexer == nil {
		return nil
	}
	source := strings.Join(lines, "\n")
	iter, err := lexer.Tokenise(nil, source)
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	if err := r.formatter.Format(&buf, r.light, iter); err != nil {
		return nil
	}
	return extractChromaLines(buf.String(), len(lines))
}

func (r *Renderer) BuildCSS() string {
	var buf bytes.Buffer
	_ = r.formatter.WriteCSS(&buf, r.light)
	lightCSS := scopeChromaCSS(buf.String(), "body.theme-light ")

	buf.Reset()
	_ = r.formatter.WriteCSS(&buf, r.dark)
	darkCSS := scopeChromaCSS(buf.String(), "body.theme-dark ")

	return lightCSS + "\n" + darkCSS + "\n"
}

func resolveLexer(path string, lines []string) chroma.Lexer {
	lexer := lexers.Match(path)
	if lexer == nil {
		joined := strings.Join(lines, "\n")
		lexer = lexers.Analyse(joined)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	return chroma.Coalesce(lexer)
}

func extractChromaLines(htmlIn string, expected int) []template.HTML {
	const lineOpen = `<span class="line">`
	const clOpen = `<span class="cl">`
	const lineClose = `</span></span>`

	lines := make([]template.HTML, 0, expected)
	remaining := htmlIn
	for len(lines) < expected {
		lineIdx := strings.Index(remaining, lineOpen)
		if lineIdx == -1 {
			break
		}
		remaining = remaining[lineIdx+len(lineOpen):]
		clIdx := strings.Index(remaining, clOpen)
		if clIdx == -1 {
			break
		}
		remaining = remaining[clIdx+len(clOpen):]
		endIdx := strings.Index(remaining, lineClose)
		if endIdx == -1 {
			break
		}
		lineHTML := fixWhitespaceSpans(remaining[:endIdx])
		lineHTML = normalizeTextWhitespace(lineHTML)
		if strings.TrimSpace(lineHTML) == "" {
			lineHTML = "&nbsp;"
		}
		lines = append(lines, template.HTML(`<span class="chroma">`+lineHTML+`</span>`))
		remaining = remaining[endIdx+len(lineClose):]
	}
	return lines
}

var whitespaceSpanRE = regexp.MustCompile(`<span class="w">(.*?)</span>`)

func fixWhitespaceSpans(input string) string {
	return whitespaceSpanRE.ReplaceAllStringFunc(input, func(match string) string {
		sub := whitespaceSpanRE.FindStringSubmatch(match)
		if len(sub) != 2 {
			return match
		}
		content := sub[1]
		content = strings.ReplaceAll(content, "\t", "&nbsp;&nbsp;&nbsp;&nbsp;")
		content = strings.ReplaceAll(content, " ", "&nbsp;")
		if content == "" {
			content = "&nbsp;"
		}
		content += "&#8203;"
		return `<span class="w">` + content + `</span>`
	})
}

func scopeChromaCSS(input, prefix string) string {
	return strings.ReplaceAll(input, ".chroma", prefix+".chroma")
}

// EscapePlain is exposed for tests and fallbacks.
func EscapePlain(s string) template.HTML {
	return template.HTML(html.EscapeString(s))
}

func normalizeTextWhitespace(input string) string {
	var out strings.Builder
	out.Grow(len(input) + 32)

	inTag := false
	var textBuf strings.Builder

	flushText := func() {
		if textBuf.Len() == 0 {
			return
		}
		segment := textBuf.String()
		textBuf.Reset()
		var segOut strings.Builder
		segOut.Grow(len(segment) * 2)
		onlyWhitespace := true
		for _, r := range segment {
			switch r {
			case ' ':
				segOut.WriteString("&nbsp;")
			case '\t':
				segOut.WriteString("&nbsp;&nbsp;&nbsp;&nbsp;")
			case '\n', '\r':
				// skip newlines within a line
			default:
				onlyWhitespace = false
				segOut.WriteRune(r)
			}
		}
		if onlyWhitespace {
			segOut.WriteString("&#8203;")
		}
		out.WriteString(segOut.String())
	}

	for _, r := range input {
		switch r {
		case '<':
			if !inTag {
				flushText()
				inTag = true
			}
			out.WriteRune(r)
		case '>':
			out.WriteRune(r)
			if inTag {
				inTag = false
			}
		default:
			if inTag {
				out.WriteRune(r)
			} else {
				textBuf.WriteRune(r)
			}
		}
	}
	flushText()
	return out.String()
}
