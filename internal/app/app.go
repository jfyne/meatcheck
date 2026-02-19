package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/jfyne/live"
	"github.com/jfyne/meatcheck/internal/highlight"
	"github.com/jfyne/meatcheck/internal/ui"
	"github.com/pkg/browser"
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

type Comment struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Text      string `json:"text"`
}

type File struct {
	Path      string
	PathSlash string
	Lines     []string
}

type TreeItem struct {
	Name     string
	Path     string
	Depth    int
	IsDir    bool
	Selected bool
}

type ViewLine struct {
	Number    int
	Text      string
	HTML      template.HTML
	Selected  bool
	Commented bool
	Comments  []ViewComment
}

type ViewFile struct {
	Path             string
	Lines            []ViewLine
	MarkdownFile     bool
	MarkdownRendered bool
	MarkdownHTML     template.HTML
}

type ViewMode string

const (
	ModeFile ViewMode = "file"
	ModeDiff ViewMode = "diff"
)

type ViewDiffLine struct {
	Kind      DiffLineKind
	OldLine   int
	NewLine   int
	Text      string
	HTML      template.HTML
	Selected  bool
	Commented bool
	Comments  []ViewComment
}

type ViewDiffHunk struct {
	Header string
	Lines  []ViewDiffLine
}

type ViewDiffFile struct {
	Path  string
	Hunks []ViewDiffHunk
}

type ViewComment struct {
	Comment
	Rendered template.HTML
}

type LineRange struct {
	Start int
	End   int
}

type ReviewModel struct {
	Files          []File
	DiffFiles      []DiffFile
	Tree           []TreeItem
	SelectedPath   string
	SelectedLabel  string
	CodeViewKey    string
	Mode           ViewMode
	RenderFile     bool
	RenderComments bool
	Prompt         string
	PromptHTML     template.HTML
	SelectionStart int
	SelectionEnd   int
	CommentDraft   string
	Comments       []Comment
	Ranges         map[string][]LineRange
	ViewFile       ViewFile
	ViewDiff       ViewDiffFile
	Error          string
}

type ReviewServer struct {
	Model    *ReviewModel
	DoneCh   chan struct{}
	DoneOnce sync.Once
}

type Config struct {
	Host    string
	Port    int
	Paths   []string
	Prompt  string
	Diff    string
	Ranges  map[string][]LineRange
	StdDiff string
}

func PrintHelp(w io.Writer) {
	fmt.Fprint(w, `meatcheck - local PR-style review UI

Usage:
  meatcheck [--host 127.0.0.1] [--port 0] <file1> <file2> ...
  meatcheck --diff <diff-file>
  meatcheck --diff <diff-file> --prompt "Review the changes"

Flags:
  --host   host to bind (default 127.0.0.1)
  --port   port to bind, 0 = random free port (default 0)
  --prompt review prompt/question to display at top
  --diff   path to unified diff file (or pipe via stdin)
  --range  file section to render (path:start-end), repeatable
  --help   show this help and exit
  --skill  print agent skill markdown and exit
`)
}

func Run(ctx context.Context, cfg Config) error {
	diffInput := strings.TrimSpace(cfg.StdDiff)
	if cfg.Diff != "" {
		data, err := os.ReadFile(cfg.Diff)
		if err != nil {
			return fmt.Errorf("read diff: %w", err)
		}
		diffInput = string(data)
	}

	var files []File
	var diffFiles []DiffFile
	mode := ModeFile
	if diffInput != "" {
		parsed, err := parseUnifiedDiff(diffInput)
		if err != nil {
			return err
		}
		if len(parsed) == 0 {
			return errors.New("no files in diff")
		}
		diffFiles = parsed
		mode = ModeDiff
	} else {
		if len(cfg.Paths) == 0 {
			return errors.New("no files provided")
		}
		loaded, err := loadFiles(cfg.Paths)
		if err != nil {
			return err
		}
		files = loaded
	}

	model := &ReviewModel{
		Files:          files,
		DiffFiles:      diffFiles,
		SelectedPath:   "",
		SelectedLabel:  "",
		Mode:           mode,
		RenderFile:     true,
		RenderComments: true,
		Prompt:         cfg.Prompt,
		Ranges:         cfg.Ranges,
	}
	if strings.TrimSpace(cfg.Prompt) != "" {
		model.PromptHTML = renderMarkdown(cfg.Prompt)
	}
	model.CodeViewKey = fmt.Sprintf("%d", time.Now().UnixNano())
	if mode == ModeDiff {
		model.SelectedPath = diffFiles[0].Path
		model.Tree = buildTree(diffFilesAsFiles(diffFiles), model.SelectedPath)
	} else {
		model.SelectedPath = files[0].Path
		model.Tree = buildTree(files, model.SelectedPath)
	}
	updateView(model)

	meatcheckServer := &ReviewServer{
		Model:  model,
		DoneCh: make(chan struct{}),
	}

	h := buildLiveHandler(meatcheckServer)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		return err
	}
	addr := listener.Addr().String()

	mux := http.NewServeMux()
	mux.Handle("/live.js", live.Javascript{})
	mux.Handle("/", live.NewHttpHandler(ctx, h))

	srv := &http.Server{Handler: mux}

	go func() {
		_ = srv.Serve(listener)
	}()

	urlStr := fmt.Sprintf("http://%s/", addr)
	if err := browser.OpenURL(urlStr); err != nil {
		fmt.Fprintf(os.Stderr, "open this URL in your browser: %s\n", urlStr)
	}

	<-meatcheckServer.DoneCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	_ = srv.Shutdown(shutdownCtx)
	cancel()

	if err := emitToon(os.Stdout, meatcheckServer.Model.Comments); err != nil {
		return err
	}
	return nil
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

func loadFiles(paths []string) ([]File, error) {
	files := make([]File, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		files = append(files, File{
			Path:      path,
			PathSlash: filepath.ToSlash(path),
			Lines:     lines,
		})
	}
	return files, nil
}

func ReadStdDiff() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}
	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder
	for {
		chunk, err := reader.ReadString('\n')
		b.WriteString(chunk)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
	}
	return b.String(), nil
}

func ParseRangeFlag(values []string) (map[string][]LineRange, error) {
	if len(values) == 0 {
		return nil, nil
	}
	ranges := make(map[string][]LineRange)
	for _, val := range values {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		parts := strings.SplitN(val, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range: %s", val)
		}
		path := parts[0]
		r := parts[1]
		seg := strings.SplitN(r, "-", 2)
		if len(seg) != 2 {
			return nil, fmt.Errorf("invalid range: %s", val)
		}
		start := mustAtoi(seg[0])
		end := mustAtoi(seg[1])
		if start == 0 || end == 0 {
			return nil, fmt.Errorf("invalid range: %s", val)
		}
		if end < start {
			start, end = end, start
		}
		ranges[path] = append(ranges[path], LineRange{Start: start, End: end})
	}
	return ranges, nil
}

func buildLiveHandler(rs *ReviewServer) *live.Handler {
	tmpl := template.Must(template.New("meatcheck").Funcs(template.FuncMap{
		"mul": func(a, b int) int {
			return a * b
		},
		"id": func(s string) string {
			var b strings.Builder
			for _, r := range s {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					b.WriteRune(r)
				} else {
					b.WriteByte('-')
				}
			}
			return b.String()
		},
	}).Parse(templateHTML))

	h := live.NewHandler()
	h.RenderHandler = func(ctx context.Context, rc *live.RenderContext) (io.Reader, error) {
		css := buildCSS()
		logoData := template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(logoBytes))
		avatarData := template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(avatarBytes))
		data := struct {
			CSS    template.CSS
			Logo   template.URL
			Avatar template.URL
			*live.RenderContext
		}{
			CSS:           template.CSS(css),
			Logo:          logoData,
			Avatar:        avatarData,
			RenderContext: rc,
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, err
		}
		return &buf, nil
	}

	h.MountHandler = func(ctx context.Context, s *live.Socket) (any, error) {
		return rs.Model, nil
	}

	h.HandleEvent("select-file", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		path := p.String("path")
		if path == "" {
			return model, nil
		}
		switch model.Mode {
		case ModeDiff:
			if hasDiffFile(model.DiffFiles, path) {
				model.SelectedPath = path
				model.CodeViewKey = fmt.Sprintf("%d", time.Now().UnixNano())
				model.SelectionStart = 0
				model.SelectionEnd = 0
				model.Error = ""
				model.Tree = buildTree(diffFilesAsFiles(model.DiffFiles), model.SelectedPath)
				updateView(model)
			}
		default:
			if hasFile(model.Files, path) {
				model.SelectedPath = path
				model.CodeViewKey = fmt.Sprintf("%d", time.Now().UnixNano())
				model.SelectionStart = 0
				model.SelectionEnd = 0
				model.Error = ""
				model.Tree = buildTree(model.Files, model.SelectedPath)
				updateView(model)
			}
		}
		return model, nil
	})

	h.HandleEvent("toggle-file-render", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		if model.Mode == ModeFile && isMarkdownPath(model.SelectedPath) {
			model.ViewFile.MarkdownRendered = !model.ViewFile.MarkdownRendered
		} else {
			model.RenderFile = !model.RenderFile
		}
		updateView(model)
		return model, nil
	})

	h.HandleEvent("toggle-comment-render", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		model.RenderComments = !model.RenderComments
		updateView(model)
		return model, nil
	})

	h.HandleEvent("select-line", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		line := p.Int("line")
		shift := p.String("shift") == "1"
		if line <= 0 {
			return model, nil
		}
		if model.Mode == ModeDiff {
			if !diffLineExists(model.DiffFiles, model.SelectedPath, line) {
				return model, nil
			}
		}
		if shift && model.SelectionStart > 0 {
			start := model.SelectionStart
			end := line
			if end < start {
				start, end = end, start
			}
			model.SelectionStart = start
			model.SelectionEnd = end
		} else {
			model.SelectionStart = line
			model.SelectionEnd = line
		}
		model.Error = ""
		updateView(model)
		return model, nil
	})

	h.HandleEvent("add-comment", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		text := strings.TrimSpace(p.String("comment"))
		if text == "" {
			model.Error = "comment text is required"
			return model, nil
		}
		if model.SelectionStart == 0 || model.SelectionEnd == 0 {
			model.Error = "select a line or range first"
			return model, nil
		}
		model.Comments = append(model.Comments, Comment{
			Path:      model.SelectedPath,
			StartLine: model.SelectionStart,
			EndLine:   model.SelectionEnd,
			Text:      text,
		})
		model.CommentDraft = ""
		model.Error = ""
		model.SelectionStart = 0
		model.SelectionEnd = 0
		updateView(model)
		return model, nil
	})

	h.HandleEvent("cancel-comment", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		model.CommentDraft = ""
		model.Error = ""
		model.SelectionStart = 0
		model.SelectionEnd = 0
		updateView(model)
		return model, nil
	})

	h.HandleEvent("finish", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		if s != nil {
			_ = s.Send("close-tab", map[string]any{})
			about, _ := url.Parse("about:blank")
			if about != nil {
				s.Redirect(about)
			}
		}
		rs.DoneOnce.Do(func() {
			close(rs.DoneCh)
		})
		return model, nil
	})

	return h
}

func getModel(s *live.Socket, fallback *ReviewModel) *ReviewModel {
	if s == nil {
		return fallback
	}
	if assigns := s.Assigns(); assigns != nil {
		if model, ok := assigns.(*ReviewModel); ok {
			return model
		}
	}
	return fallback
}

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
			if model.ViewFile.Path != selectedFile.Path {
				viewFile.MarkdownRendered = true
			} else {
				viewFile.MarkdownRendered = model.ViewFile.MarkdownRendered
			}
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
	commented := false
	var lineComments []ViewComment
	for _, c := range comments {
		if c.Path != file.Path {
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
			var lineComments []ViewComment
			for _, c := range comments {
				if c.Path != file.Path {
					continue
				}
				if dl.NewLine > 0 && dl.NewLine >= c.StartLine && dl.NewLine <= c.EndLine {
					line.Commented = true
				}
				if dl.NewLine > 0 && dl.NewLine == c.StartLine {
					lineComments = append(lineComments, ViewComment{
						Comment:  c,
						Rendered: renderMarkdown(c.Text),
					})
				}
			}
			line.Comments = lineComments
			if !selectable {
				line.Selected = false
			}
			vh.Lines = append(vh.Lines, line)
		}
		view.Hunks = append(view.Hunks, vh)
	}
	return view
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

func buildTree(files []File, selectedPath string) []TreeItem {
	root := &treeNode{Name: "", Path: "", IsDir: true, Children: map[string]*treeNode{}}
	for i := range files {
		pathSlash := files[i].PathSlash
		parts := strings.Split(pathSlash, "/")
		cur := root
		for j := 0; j < len(parts)-1; j++ {
			name := parts[j]
			if name == "" {
				continue
			}
			next, ok := cur.Children[name]
			if !ok {
				next = &treeNode{Name: name, Path: joinPath(cur.Path, name), IsDir: true, Children: map[string]*treeNode{}}
				cur.Children[name] = next
			}
			cur = next
		}
		fileName := parts[len(parts)-1]
		node := &treeNode{Name: fileName, Path: pathSlash, IsDir: false, File: &files[i]}
		cur.Children[fileName] = node
	}

	var items []TreeItem
	var walk func(n *treeNode, depth int)
	walk = func(n *treeNode, depth int) {
		if n != root {
			item := TreeItem{
				Name:     n.Name,
				Path:     "",
				Depth:    depth,
				IsDir:    n.IsDir,
				Selected: n.File != nil && n.File.Path == selectedPath,
			}
			if n.File != nil {
				item.Path = n.File.Path
			}
			items = append(items, item)
		}
		children := make([]*treeNode, 0, len(n.Children))
		for _, child := range n.Children {
			children = append(children, child)
		}
		sort.Slice(children, func(i, j int) bool {
			if children[i].IsDir != children[j].IsDir {
				return children[i].IsDir
			}
			return children[i].Name < children[j].Name
		})
		for _, child := range children {
			walk(child, depth+1)
		}
	}
	walk(root, -1)
	return items
}

type treeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children map[string]*treeNode
	File     *File
}

func joinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func hasFile(files []File, path string) bool {
	for _, f := range files {
		if f.Path == path {
			return true
		}
	}
	return false
}

func findFile(files []File, path string) *File {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}

func emitToon(w *os.File, comments []Comment) error {
	doc := map[string]any{
		"comments": comments,
	}
	encoded, err := gotoon.Encode(doc)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, encoded)
	return err
}

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
