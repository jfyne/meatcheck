package main

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"fmt"
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
	"github.com/pkg/browser"
)

//go:embed template.html styles.css
var embeddedFiles embed.FS

var (
	templateHTML = mustReadEmbedded("template.html")
	stylesCSS    = mustReadEmbedded("styles.css")
)

func mustReadEmbedded(path string) string {
	data, err := embeddedFiles.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

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
	Selected  bool
	Commented bool
	Comments  []Comment
}

type ViewFile struct {
	Path  string
	Lines []ViewLine
}

type ReviewModel struct {
	Files          []File
	Tree           []TreeItem
	SelectedPath   string
	CodeViewKey    string
	SelectionStart int
	SelectionEnd   int
	CommentDraft   string
	Comments       []Comment
	ViewFile       ViewFile
	Error          string
}

type ReviewServer struct {
	Model    *ReviewModel
	DoneCh   chan struct{}
	DoneOnce sync.Once
}

func main() {
	var (
		host = flag.String("host", "127.0.0.1", "host to bind")
		port = flag.Int("port", 0, "port to bind (0 = random)")
	)
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: review <file1> <file2> ...")
		os.Exit(2)
	}

	files, err := loadFiles(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	model := &ReviewModel{
		Files:        files,
		SelectedPath: files[0].Path,
	}
	model.CodeViewKey = fmt.Sprintf("%d", time.Now().UnixNano())
	model.Tree = buildTree(files, model.SelectedPath)
	updateView(model)

	reviewServer := &ReviewServer{
		Model:  model,
		DoneCh: make(chan struct{}),
	}

	h := buildLiveHandler(reviewServer)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	addr := listener.Addr().String()

	mux := http.NewServeMux()
	mux.Handle("/live.js", live.Javascript{})
	mux.Handle("/", live.NewHttpHandler(context.Background(), h))

	srv := &http.Server{Handler: mux}

	go func() {
		_ = srv.Serve(listener)
	}()

	url := fmt.Sprintf("http://%s/", addr)
	if err := browser.OpenURL(url); err != nil {
		fmt.Fprintf(os.Stderr, "open this URL in your browser: %s\n", url)
	}

	<-reviewServer.DoneCh

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	_ = srv.Shutdown(ctx)
	cancel()

	if err := emitToon(os.Stdout, reviewServer.Model.Comments); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
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

func buildLiveHandler(rs *ReviewServer) *live.Handler {
	tmpl := template.Must(template.New("review").Funcs(template.FuncMap{
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
		data := struct {
			CSS template.CSS
			*live.RenderContext
		}{
			CSS:           template.CSS(stylesCSS),
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
		if hasFile(model.Files, path) {
			model.SelectedPath = path
			model.CodeViewKey = fmt.Sprintf("%d", time.Now().UnixNano())
			model.SelectionStart = 0
			model.SelectionEnd = 0
			model.Error = ""
			model.Tree = buildTree(model.Files, model.SelectedPath)
			updateView(model)
		}
		return model, nil
	})

	h.HandleEvent("select-line", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		line := p.Int("line")
		shift := p.String("shift") == "1"
		if line <= 0 {
			return model, nil
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
	selectedFile := findFile(model.Files, model.SelectedPath)
	viewFile := ViewFile{Path: model.SelectedPath}
	if selectedFile != nil {
		viewFile.Lines = buildViewLines(selectedFile, model.Comments, model.SelectionStart, model.SelectionEnd)
	}
	model.ViewFile = viewFile
}

func buildViewLines(file *File, comments []Comment, start, end int) []ViewLine {
	lines := make([]ViewLine, 0, len(file.Lines))
	for i, raw := range file.Lines {
		lineNum := i + 1
		selected := start > 0 && end > 0 && lineNum >= start && lineNum <= end
		commented := false
		var lineComments []Comment
		for _, c := range comments {
			if c.Path != file.Path {
				continue
			}
			if lineNum >= c.StartLine && lineNum <= c.EndLine {
				commented = true
			}
			if lineNum == c.StartLine {
				lineComments = append(lineComments, c)
			}
		}
		lines = append(lines, ViewLine{
			Number:    lineNum,
			Text:      raw,
			Selected:  selected,
			Commented: commented,
			Comments:  lineComments,
		})
	}
	return lines
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
