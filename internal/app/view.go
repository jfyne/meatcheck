package app

import (
	"fmt"
	"html/template"
	"path/filepath"
	"sort"
	"strings"
)

func updateView(model *ReviewModel) {
	switch model.Mode {
	case ModeDiff:
		updateDiffView(model)
	default:
		updateFileView(model)
	}
}

func updateFileView(model *ReviewModel) {
	selectedFile := findFile(model.Files, model.SelectedPath)
	viewFile := ViewFile{Path: model.SelectedPath}
	if selectedFile != nil {
		viewFile.MarkdownFile = isMarkdownPath(selectedFile.Path)
		if viewFile.MarkdownFile {
			if model.MarkdownRenderByPath == nil {
				model.MarkdownRenderByPath = make(map[string]bool)
			}
			rendered, ok := model.MarkdownRenderByPath[selectedFile.Path]
			if !ok {
				rendered = true
				model.MarkdownRenderByPath[selectedFile.Path] = true
			}
			viewFile.MarkdownRendered = rendered
		}
		if viewFile.MarkdownFile && viewFile.MarkdownRendered {
			viewFile.MarkdownHTML = renderMarkdown(strings.Join(selectedFile.Lines, "\n"))
			model.ViewFile = viewFile
			model.SelectedLabel = formatSelectedLabel(model.SelectedPath, model.Ranges[model.SelectedPath])
			return
		}
		var rendered []template.HTML
		if model.RenderFile {
			rendered = codeRenderer.RenderLines(selectedFile.Path, selectedFile.Lines)
		}
		viewFile.Lines = buildViewLinesWithRanges(selectedFile, model.Comments, model.SelectionStart, model.SelectionEnd, rendered, model.Ranges[selectedFile.Path])
	}
	model.ViewFile = viewFile
	model.SelectedLabel = formatSelectedLabel(model.SelectedPath, model.Ranges[model.SelectedPath])
}

func updateDiffView(model *ReviewModel) {
	diffFile := findDiffFile(model.DiffFiles, model.SelectedPath)
	viewDiff := ViewDiffFile{Path: model.SelectedPath}
	if diffFile != nil {
		viewDiff = buildViewDiff(diffFile, model.Comments, model.SelectionStart, model.SelectionEnd, model.RenderFile)
	}
	model.ViewDiff = viewDiff
	model.SelectedLabel = model.SelectedPath
}

func buildViewLinesWithRanges(file *File, comments []Comment, start, end int, rendered []template.HTML, ranges []LineRange) []ViewLine {
	if len(ranges) == 0 {
		return buildViewLines(file, comments, start, end, rendered)
	}
	norm := normalizeRanges(ranges)
	lines := make([]ViewLine, 0, len(file.Lines))
	for _, r := range norm {
		if r.Start < 1 {
			r.Start = 1
		}
		if r.End > len(file.Lines) {
			r.End = len(file.Lines)
		}
		for i := r.Start - 1; i < r.End; i++ {
			lines = append(lines, buildSingleViewLine(file, comments, start, end, rendered, i))
		}
	}
	return lines
}

func buildSingleViewLine(file *File, comments []Comment, start, end int, rendered []template.HTML, idx int) ViewLine {
	lineNum := idx + 1
	raw := file.Lines[idx]
	selected := start > 0 && end > 0 && lineNum >= start && lineNum <= end
	commented, lineComments := projectLineComments(file.Path, lineNum, comments)

	lineHTML := template.HTML("")
	if len(rendered) > idx {
		lineHTML = rendered[idx]
	}
	return ViewLine{
		Number:    lineNum,
		Text:      raw,
		HTML:      lineHTML,
		Selected:  selected,
		Commented: commented,
		Comments:  lineComments,
	}
}

func buildViewLines(file *File, comments []Comment, start, end int, rendered []template.HTML) []ViewLine {
	lines := make([]ViewLine, 0, len(file.Lines))
	for i := range file.Lines {
		lines = append(lines, buildSingleViewLine(file, comments, start, end, rendered, i))
	}
	return lines
}

func buildViewDiff(file *DiffFile, comments []Comment, start, end int, render bool) ViewDiffFile {
	view := ViewDiffFile{Path: file.Path}
	for _, h := range file.Hunks {
		hdr := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		vh := ViewDiffHunk{Header: hdr}
		var rendered []template.HTML
		if render {
			lines := make([]string, 0, len(h.Lines))
			for _, dl := range h.Lines {
				lines = append(lines, dl.Text)
			}
			rendered = codeRenderer.RenderLines(file.Path, lines)
		}
		for i, dl := range h.Lines {
			line := ViewDiffLine{
				Kind:    dl.Kind,
				OldLine: dl.OldLine,
				NewLine: dl.NewLine,
				Text:    dl.Text,
			}
			if len(rendered) > i {
				line.HTML = rendered[i]
			}
			selectable := dl.NewLine > 0 && dl.Kind != DiffDel
			if selectable && start > 0 && end > 0 && dl.NewLine >= start && dl.NewLine <= end {
				line.Selected = true
			}
			if dl.NewLine > 0 {
				line.Commented, line.Comments = projectLineComments(file.Path, dl.NewLine, comments)
			}
			if !selectable {
				line.Selected = false
			}
			vh.Lines = append(vh.Lines, line)
		}
		view.Hunks = append(view.Hunks, vh)
	}
	return view
}

func projectLineComments(path string, lineNum int, comments []Comment) (bool, []ViewComment) {
	commented := false
	lineComments := make([]ViewComment, 0)
	for _, c := range comments {
		if c.Path != path {
			continue
		}
		if lineNum >= c.StartLine && lineNum <= c.EndLine {
			commented = true
		}
		if lineNum == c.StartLine {
			lineComments = append(lineComments, ViewComment{
				Comment:  c,
				Rendered: renderMarkdown(c.Text),
			})
		}
	}
	return commented, lineComments
}

func diffFilesAsFiles(diffFiles []DiffFile) []File {
	files := make([]File, 0, len(diffFiles))
	for _, df := range diffFiles {
		files = append(files, File{Path: df.Path, PathSlash: filepath.ToSlash(df.Path)})
	}
	return files
}

func findDiffFile(files []DiffFile, path string) *DiffFile {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}

func hasDiffFile(files []DiffFile, path string) bool {
	return findDiffFile(files, path) != nil
}

func diffLineExists(files []DiffFile, path string, line int) bool {
	file := findDiffFile(files, path)
	if file == nil {
		return false
	}
	for _, h := range file.Hunks {
		for _, dl := range h.Lines {
			if dl.NewLine == line && dl.Kind != DiffDel {
				return true
			}
		}
	}
	return false
}

func normalizeRanges(ranges []LineRange) []LineRange {
	if len(ranges) == 0 {
		return nil
	}
	var cleaned []LineRange
	for _, r := range ranges {
		if r.Start <= 0 || r.End <= 0 {
			continue
		}
		if r.End < r.Start {
			r.Start, r.End = r.End, r.Start
		}
		cleaned = append(cleaned, r)
	}
	if len(cleaned) == 0 {
		return nil
	}
	sort.Slice(cleaned, func(i, j int) bool {
		if cleaned[i].Start == cleaned[j].Start {
			return cleaned[i].End < cleaned[j].End
		}
		return cleaned[i].Start < cleaned[j].Start
	})
	merged := []LineRange{cleaned[0]}
	for _, r := range cleaned[1:] {
		last := &merged[len(merged)-1]
		if r.Start <= last.End+1 {
			if r.End > last.End {
				last.End = r.End
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged
}

func formatSelectedLabel(path string, ranges []LineRange) string {
	if len(ranges) == 0 {
		return path
	}
	norm := normalizeRanges(ranges)
	parts := make([]string, 0, len(norm))
	for _, r := range norm {
		parts = append(parts, fmt.Sprintf("%d-%d", r.Start, r.End))
	}
	return fmt.Sprintf("%s (lines %s)", path, strings.Join(parts, ", "))
}
