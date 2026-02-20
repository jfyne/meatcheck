package app

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jfyne/live"
)

func TestHTTPRenderIncludesThemeClass(t *testing.T) {
	model := &ReviewModel{
		Files:                []File{{Path: "a.go", PathSlash: "a.go", Lines: []string{"package main"}}},
		SelectedPath:         "a.go",
		Mode:                 ModeFile,
		RenderFile:           true,
		RenderComments:       true,
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)
	if !strings.Contains(html, `<body class="theme-dark">`) {
		t.Fatalf("expected rendered html to include theme class for chroma css, got: %q", html)
	}
}

func TestHTTPRenderFileModeCommentFormAutofocus(t *testing.T) {
	model := &ReviewModel{
		Files: []File{{
			Path:      "a.go",
			PathSlash: "a.go",
			Lines:     []string{"package main", "func main() {}"},
		}},
		SelectedPath:         "a.go",
		SelectionStart:       2,
		SelectionEnd:         2,
		Mode:                 ModeFile,
		RenderFile:           true,
		RenderComments:       true,
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(model.Files, model.SelectedPath)

	html := renderReviewHTML(t, model)
	if !strings.Contains(html, `<textarea name="comment" placeholder="Leave a comment..." autofocus></textarea>`) {
		t.Fatalf("expected comment textarea autofocus in file mode, got: %q", html)
	}
}

func TestHTTPRenderDiffModeCommentFormAutofocus(t *testing.T) {
	model := &ReviewModel{
		DiffFiles: []DiffFile{{
			Path: "a.go",
			Hunks: []DiffHunk{{
				OldStart: 1,
				OldCount: 1,
				NewStart: 1,
				NewCount: 1,
				Lines: []DiffLine{
					{Kind: DiffContext, OldLine: 1, NewLine: 1, Text: "package main"},
				},
			}},
		}},
		SelectedPath:         "a.go",
		SelectionStart:       1,
		SelectionEnd:         1,
		Mode:                 ModeDiff,
		RenderFile:           true,
		RenderComments:       true,
		Ranges:               map[string][]LineRange{},
		MarkdownRenderByPath: map[string]bool{},
	}
	model.Tree = buildTree(diffFilesAsFiles(model.DiffFiles), model.SelectedPath)

	html := renderReviewHTML(t, model)
	if !strings.Contains(html, `<textarea name="comment" placeholder="Leave a comment..." autofocus></textarea>`) {
		t.Fatalf("expected comment textarea autofocus in diff mode, got: %q", html)
	}
}

func renderReviewHTML(t *testing.T, model *ReviewModel) string {
	t.Helper()

	updateView(model)

	rs := &ReviewServer{Model: model, DoneCh: make(chan struct{})}
	h := buildLiveHandler(rs)
	out, err := h.RenderHandler(context.Background(), &live.RenderContext{Assigns: model})
	if err != nil {
		t.Fatalf("render handler failed: %v", err)
	}
	b, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("read render output failed: %v", err)
	}
	return string(b)
}
