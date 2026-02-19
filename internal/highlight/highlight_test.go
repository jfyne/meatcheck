package highlight

import (
	"strings"
	"testing"
)

func TestBuildCSSScopesThemes(t *testing.T) {
	r := NewRenderer("github", "dracula", 4)
	css := r.BuildCSS()
	if !strings.Contains(css, "body.theme-light .chroma") {
		t.Fatalf("expected light theme chroma scope in CSS")
	}
	if !strings.Contains(css, "body.theme-dark .chroma") {
		t.Fatalf("expected dark theme chroma scope in CSS")
	}
}

func TestRenderLinesPreservesWhitespace(t *testing.T) {
	r := NewRenderer("github", "dracula", 4)
	lines := []string{"\tfoo", "  bar"}
	rendered := r.RenderLines("test.go", lines)
	if len(rendered) != len(lines) {
		t.Fatalf("expected %d rendered lines, got %d", len(lines), len(rendered))
	}
	first := string(rendered[0])
	if !strings.Contains(first, "&nbsp;") {
		t.Fatalf("expected rendered line to contain nbsp for whitespace, got: %s", first)
	}
	if !strings.Contains(first, "&#8203;") {
		t.Fatalf("expected rendered line to contain zero-width space to prevent trimming")
	}
	second := string(rendered[1])
	if !strings.Contains(second, "&nbsp;") {
		t.Fatalf("expected rendered line to contain nbsp for whitespace, got: %s", second)
	}
}

func TestRenderLinesPreservesLeadingSpaces(t *testing.T) {
	r := NewRenderer("github", "dracula", 4)
	lines := []string{"    spaced", "  double", "\tindented"}
	rendered := r.RenderLines("test.go", lines)
	if len(rendered) != len(lines) {
		t.Fatalf("expected %d rendered lines, got %d", len(lines), len(rendered))
	}
	for i, line := range rendered {
		if !strings.Contains(string(line), "&nbsp;") {
			t.Fatalf("expected line %d to include nbsp for spaces, got: %s", i, string(line))
		}
	}
}
