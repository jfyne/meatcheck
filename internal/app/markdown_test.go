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
	if !strings.Contains(string(m.ViewFile.MarkdownHTML), "<h1>") {
		t.Fatalf("expected rendered markdown HTML, got %q", string(m.ViewFile.MarkdownHTML))
	}
	if len(m.ViewFile.Lines) != 0 {
		t.Fatal("did not expect code lines in markdown rendered mode")
	}
}

func TestUpdateFileViewMarkdownCodeMode(t *testing.T) {
	m := &ReviewModel{
		Files:        []File{{Path: "README.md", PathSlash: "README.md", Lines: []string{"# Heading", "Hello"}}},
		SelectedPath: "README.md",
		RenderFile:   true,
		ViewFile: ViewFile{
			Path:             "README.md",
			MarkdownRendered: false,
		},
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
