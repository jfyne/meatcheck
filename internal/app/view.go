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
			blocks := renderMarkdownBlocks(selectedFile.Path, strings.Join(selectedFile.Lines, "\n"))
			for i := range blocks {
				blocks[i].Selected = model.SelectionStart > 0 && model.SelectionEnd > 0 &&
					blocks[i].EndLine >= model.SelectionStart && blocks[i].StartLine <= model.SelectionEnd
				blocks[i].Commented, blocks[i].Comments = projectBlockComments(
					selectedFile.Path, blocks[i].StartLine, blocks[i].EndLine,
					model.Comments, model.EditingCommentID,
				)
			}
			viewFile.MarkdownBlocks = blocks
			model.ViewFile = viewFile
			model.SelectedLabel = formatSelectedLabel(model.SelectedPath, model.Ranges[model.SelectedPath])
			return
		}
		var rendered []template.HTML
		if model.RenderFile {
			rendered = codeRenderer.RenderLines(selectedFile.Path, selectedFile.Lines)
		}
		viewFile.Lines = buildViewLinesWithRanges(selectedFile, model.Comments, model.SelectionStart, model.SelectionEnd, rendered, model.Ranges[selectedFile.Path], model.EditingCommentID)
	}
	model.ViewFile = viewFile
	model.SelectedLabel = formatSelectedLabel(model.SelectedPath, model.Ranges[model.SelectedPath])
}

func updateDiffView(model *ReviewModel) {
	model.ViewDiff = ViewDiffFile{}
	model.ViewDiffSplit = nil

	diffFile := findDiffFile(model.DiffFiles, model.SelectedPath)
	if diffFile != nil {
		switch model.DiffFormat {
		case DiffFormatSplit:
			model.ViewDiffSplit = buildViewDiffSplit(diffFile, model.Comments, model.SelectionStart, model.SelectionEnd, model.RenderFile, model.EditingCommentID, model.SelectionSide)
		default:
			model.ViewDiff = buildViewDiff(diffFile, model.Comments, model.SelectionStart, model.SelectionEnd, model.RenderFile, model.EditingCommentID, model.SelectionSide)
		}
	}
	model.SelectedLabel = model.SelectedPath
}

func buildViewDiffSplit(file *DiffFile, comments []Comment, start, end int, render bool, editingID int, selectionSide string) []ViewDiffSplitHunk {
	hunks := make([]ViewDiffSplitHunk, 0, len(file.Hunks))
	for _, h := range file.Hunks {
		hdr := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		vh := ViewDiffSplitHunk{Header: hdr}

		// Render syntax highlighting for all lines in the hunk if requested.
		var rendered []template.HTML
		if render {
			texts := make([]string, 0, len(h.Lines))
			for _, dl := range h.Lines {
				texts = append(texts, dl.Text)
			}
			rendered = codeRenderer.RenderLines(file.Path, texts)
		}

		// Process lines: walk sequentially, grouping del/add blocks together.
		lines := h.Lines
		i := 0
		for i < len(lines) {
			dl := lines[i]
			if dl.Kind == DiffContext {
				// Context line: both sides populated.
				row := ViewDiffRow{
					Left: ViewDiffSide{
						Line: dl.OldLine,
						Kind: DiffContext,
						Text: dl.Text,
					},
					Right: ViewDiffSide{
						Line: dl.NewLine,
						Kind: DiffContext,
						Text: dl.Text,
					},
				}
				if len(rendered) > i {
					row.Left.HTML = rendered[i]
					row.Right.HTML = rendered[i]
				}
				vh.Rows = append(vh.Rows, row)
				i++
			} else {
				// Collect consecutive del lines, then consecutive add lines.
				var dels []int // indices into lines
				var adds []int
				for i < len(lines) && lines[i].Kind == DiffDel {
					dels = append(dels, i)
					i++
				}
				for i < len(lines) && lines[i].Kind == DiffAdd {
					adds = append(adds, i)
					i++
				}

				// Zip dels and adds 1:1.
				maxLen := max(len(dels), len(adds))
				for j := range maxLen {
					row := ViewDiffRow{}
					if j < len(dels) {
						idx := dels[j]
						row.Left = ViewDiffSide{
							Line: lines[idx].OldLine,
							Kind: DiffDel,
							Text: lines[idx].Text,
						}
						if len(rendered) > idx {
							row.Left.HTML = rendered[idx]
						}
					} else {
						row.Left = ViewDiffSide{Empty: true}
					}
					if j < len(adds) {
						idx := adds[j]
						row.Right = ViewDiffSide{
							Line: lines[idx].NewLine,
							Kind: DiffAdd,
							Text: lines[idx].Text,
						}
						if len(rendered) > idx {
							row.Right.HTML = rendered[idx]
						}
					} else {
						row.Right = ViewDiffSide{Empty: true}
					}
					// Apply intra-line word-level diff for paired del/add rows.
					if j < len(dels) && j < len(adds) {
						applyIntraLineDiff(&row.Left.HTML, &row.Right.HTML, lines[dels[j]].Text, lines[adds[j]].Text)
					}
					vh.Rows = append(vh.Rows, row)
				}
			}
		}

		// Apply selection and comment projection to each row.
		for ri := range vh.Rows {
			row := &vh.Rows[ri]

			// Selection projection.
			if start > 0 && end > 0 {
				if selectionSide == "old" {
					if !row.Left.Empty && row.Left.Line > 0 && row.Left.Line >= start && row.Left.Line <= end {
						row.Left.Selected = true
					}
				} else {
					if !row.Right.Empty && row.Right.Line > 0 && row.Right.Line >= start && row.Right.Line <= end {
						row.Right.Selected = true
					}
				}
			}

			// Comment projection: left side uses "old", right side uses "".
			if !row.Left.Empty && row.Left.Line > 0 {
				row.Left.Commented, row.Left.Comments = projectLineComments(file.Path, row.Left.Line, comments, editingID, "old")
			}
			if !row.Right.Empty && row.Right.Line > 0 {
				row.Right.Commented, row.Right.Comments = projectLineComments(file.Path, row.Right.Line, comments, editingID, "")
			}
		}

		hunks = append(hunks, vh)
	}
	return hunks
}

func buildViewLinesWithRanges(file *File, comments []Comment, start, end int, rendered []template.HTML, ranges []LineRange, editingID int) []ViewLine {
	if len(ranges) == 0 {
		return buildViewLines(file, comments, start, end, rendered, editingID)
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
			lines = append(lines, buildSingleViewLine(file, comments, start, end, rendered, i, editingID))
		}
	}
	return lines
}

func buildSingleViewLine(file *File, comments []Comment, start, end int, rendered []template.HTML, idx int, editingID int) ViewLine {
	lineNum := idx + 1
	raw := file.Lines[idx]
	selected := start > 0 && end > 0 && lineNum >= start && lineNum <= end
	commented, lineComments := projectLineComments(file.Path, lineNum, comments, editingID, "")

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

func buildViewLines(file *File, comments []Comment, start, end int, rendered []template.HTML, editingID int) []ViewLine {
	lines := make([]ViewLine, 0, len(file.Lines))
	for i := range file.Lines {
		lines = append(lines, buildSingleViewLine(file, comments, start, end, rendered, i, editingID))
	}
	return lines
}

func buildViewDiff(file *DiffFile, comments []Comment, start, end int, render bool, editingID int, selectionSide string) ViewDiffFile {
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
			// Selection: when selectionSide == "old", check OldLine; otherwise check NewLine
			if start > 0 && end > 0 {
				if selectionSide == "old" {
					if dl.OldLine > 0 && dl.OldLine >= start && dl.OldLine <= end {
						line.Selected = true
					}
				} else {
					if dl.NewLine > 0 && dl.NewLine >= start && dl.NewLine <= end {
						line.Selected = true
					}
				}
			}
			// Project comments from both sides, merge results
			if dl.NewLine > 0 {
				line.Commented, line.Comments = projectLineComments(file.Path, dl.NewLine, comments, editingID, "")
			}
			if dl.OldLine > 0 {
				oldCommented, oldComments := projectLineComments(file.Path, dl.OldLine, comments, editingID, "old")
				if oldCommented {
					line.Commented = true
				}
				line.Comments = append(line.Comments, oldComments...)
			}
			vh.Lines = append(vh.Lines, line)
		}
		// Post-pass: apply intra-line word-level diff for adjacent del/add pairs.
		for i := 0; i+1 < len(vh.Lines); i++ {
			if vh.Lines[i].Kind == DiffDel && vh.Lines[i+1].Kind == DiffAdd {
				applyIntraLineDiff(&vh.Lines[i].HTML, &vh.Lines[i+1].HTML, vh.Lines[i].Text, vh.Lines[i+1].Text)
			}
		}
		view.Hunks = append(view.Hunks, vh)
	}
	return view
}

func projectLineComments(path string, lineNum int, comments []Comment, editingID int, side string) (bool, []ViewComment) {
	commented := false
	lineComments := make([]ViewComment, 0)
	for _, c := range comments {
		if c.Path != path {
			continue
		}
		// Side-aware filtering: "" matches new-side (c.Side == ""), "old" matches old-side (c.Side == "old")
		if side == "" {
			if c.Side != "" {
				continue
			}
		} else {
			if c.Side != side {
				continue
			}
		}
		if lineNum >= c.StartLine && lineNum <= c.EndLine {
			commented = true
		}
		if lineNum == c.StartLine {
			lineComments = append(lineComments, ViewComment{
				Comment:  c,
				Rendered: renderMarkdown(c.Text),
				Editing:  c.ID == editingID,
			})
		}
	}
	return commented, lineComments
}

func projectBlockComments(path string, startLine, endLine int, comments []Comment, editingID int) (bool, []ViewComment) {
	commented := false
	blockComments := make([]ViewComment, 0)
	for _, c := range comments {
		if c.Path != path {
			continue
		}
		if c.StartLine <= endLine && c.EndLine >= startLine {
			commented = true
		}
		if c.StartLine >= startLine && c.StartLine <= endLine {
			blockComments = append(blockComments, ViewComment{
				Comment:  c,
				Rendered: renderMarkdown(c.Text),
				Editing:  c.ID == editingID,
			})
		}
	}
	return commented, blockComments
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
