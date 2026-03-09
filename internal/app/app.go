package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jfyne/live"
	"github.com/pkg/browser"
)

func PrintHelp(w io.Writer) {
	fmt.Fprint(w, `meatcheck - local PR-style review UI

Usage:
  meatcheck [--host 127.0.0.1] [--port 0] <file1> <file2> ...
  meatcheck --diff <diff-file>
  meatcheck --diff <diff-file> --prompt "Review the changes"
  meatcheck --groups groups.json <file1> <file2> ...

Flags:
  --host   host to bind (default 127.0.0.1)
  --port   port to bind, 0 = random free port (default 0)
  --prompt review prompt/question to display at top
  --diff   path to unified diff file (or pipe via stdin)
  --range  file section to render (path:start-end), repeatable
  --groups path to JSON file with ordered file groups
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
		Files:                files,
		DiffFiles:            diffFiles,
		Viewed:               make(map[string]bool),
		Groups:               cfg.Groups,
		HasGroups:            len(cfg.Groups) > 0,
		SelectedPath:         "",
		SelectedLabel:        "",
		Mode:                 mode,
		DiffFormat:           DiffFormatUnified,
		RenderFile:           true,
		RenderComments:       true,
		Prompt:               cfg.Prompt,
		Ranges:               cfg.Ranges,
		MarkdownRenderByPath: make(map[string]bool),
	}
	if strings.TrimSpace(cfg.Prompt) != "" {
		model.PromptHTML = renderMarkdown(cfg.Prompt)
	}
	model.CodeViewKey = fmt.Sprintf("%d", time.Now().UnixNano())
	if model.HasGroups {
		// Select first file from first group to respect defined order.
		if mode == ModeDiff {
			df := diffFilesAsFiles(diffFiles)
			if f := findFileBySlash(df, cfg.Groups[0].Files[0]); f != nil {
				model.SelectedPath = f.Path
			} else {
				model.SelectedPath = diffFiles[0].Path
			}
		} else {
			if f := findFileBySlash(files, cfg.Groups[0].Files[0]); f != nil {
				model.SelectedPath = f.Path
			} else {
				model.SelectedPath = files[0].Path
			}
		}
	} else if mode == ModeDiff {
		model.SelectedPath = diffFiles[0].Path
	} else {
		model.SelectedPath = files[0].Path
	}
	rebuildTree(model)
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
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	mux.Handle("/file", localFileHandler(wd))
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

func rebuildTree(model *ReviewModel) {
	var files []File
	if model.Mode == ModeDiff {
		files = diffFilesAsFiles(model.DiffFiles)
	} else {
		files = model.Files
	}
	if model.HasGroups {
		model.Tree = buildGroupedTree(model.Groups, files, model.SelectedPath, model.Viewed, model.Comments)
	} else {
		model.Tree = buildTree(files, model.SelectedPath, model.Viewed, model.Comments)
	}
}

func selectFile(model *ReviewModel, path string) {
	model.SelectedPath = path
	model.CodeViewKey = fmt.Sprintf("%d", time.Now().UnixNano())
	model.SelectionStart = 0
	model.SelectionEnd = 0
	model.Error = ""
	rebuildTree(model)
	updateView(model)
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
		"commentThreadData": func(root *ReviewModel, comments []ViewComment, logo template.URL) map[string]any {
			return map[string]any{"Root": root, "Comments": comments, "Logo": logo}
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
				selectFile(model, path)
			}
		default:
			if hasFile(model.Files, path) {
				selectFile(model, path)
			}
		}
		return model, nil
	})

	h.HandleEvent("toggle-file-render", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		if model.Mode == ModeFile && isMarkdownPath(model.SelectedPath) {
			current, ok := model.MarkdownRenderByPath[model.SelectedPath]
			if !ok {
				current = true
			}
			model.MarkdownRenderByPath[model.SelectedPath] = !current
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
		lineEnd := p.Int("line_end")
		oldLine := p.Int("old_line")
		shift := p.String("shift") == "1"
		if model.Mode == ModeDiff && oldLine > 0 {
			if !diffOldLineExists(model.DiffFiles, model.SelectedPath, oldLine) {
				return model, nil
			}
			line = oldLine
			lineEnd = oldLine
			model.SelectionSide = "old"
		} else {
			if line <= 0 {
				return model, nil
			}
			if model.Mode == ModeDiff {
				if !diffLineExists(model.DiffFiles, model.SelectedPath, line) {
					return model, nil
				}
			}
			model.SelectionSide = ""
		}
		if lineEnd < line {
			lineEnd = line
		}
		if shift && model.SelectionStart > 0 {
			start := model.SelectionStart
			end := lineEnd
			if end < start {
				start, end = end, start
			}
			model.SelectionStart = start
			model.SelectionEnd = end
		} else {
			model.SelectionStart = line
			model.SelectionEnd = lineEnd
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
		model.NextCommentID++
		model.Comments = append(model.Comments, Comment{
			ID:        model.NextCommentID,
			Path:      model.SelectedPath,
			StartLine: model.SelectionStart,
			EndLine:   model.SelectionEnd,
			Side:      model.SelectionSide,
			Text:      text,
		})
		model.CommentDraft = ""
		model.Error = ""
		model.SelectionStart = 0
		model.SelectionEnd = 0
		model.SelectionSide = ""
		rebuildTree(model)
		updateView(model)
		return model, nil
	})

	h.HandleEvent("cancel-comment", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		model.CommentDraft = ""
		model.Error = ""
		model.SelectionStart = 0
		model.SelectionEnd = 0
		model.SelectionSide = ""
		updateView(model)
		return model, nil
	})

	h.HandleEvent("start-edit-comment", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		model.EditingCommentID = p.Int("id")
		model.Error = ""
		updateView(model)
		return model, nil
	})

	h.HandleEvent("edit-comment", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		id := p.Int("id")
		text := strings.TrimSpace(p.String("comment"))
		if err := editComment(model, id, text); err != nil {
			model.Error = err.Error()
			return model, nil
		}
		model.Error = ""
		rebuildTree(model)
		updateView(model)
		return model, nil
	})

	h.HandleEvent("delete-comment", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		id := p.Int("id")
		deleteComment(model, id)
		model.Error = ""
		rebuildTree(model)
		updateView(model)
		return model, nil
	})

	h.HandleEvent("cancel-edit-comment", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		model.EditingCommentID = 0
		model.Error = ""
		updateView(model)
		return model, nil
	})

	h.HandleEvent("toggle-sidebar", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		model.SidebarCollapsed = !model.SidebarCollapsed
		return model, nil
	})

	h.HandleEvent("mark-viewed", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		wasViewed := model.Viewed[model.SelectedPath]
		model.Viewed[model.SelectedPath] = !wasViewed
		if !wasViewed {
			// Just marked as viewed — advance to next unviewed
			next := nextUnviewedFile(model)
			if next != "" {
				selectFile(model, next)
			} else {
				rebuildTree(model)
				updateView(model)
			}
		} else {
			// Unmarked — stay on current file
			rebuildTree(model)
			updateView(model)
		}
		return model, nil
	})

	h.HandleEvent("toggle-diff-format", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		if model.DiffFormat == DiffFormatSplit {
			model.DiffFormat = DiffFormatUnified
		} else {
			model.DiffFormat = DiffFormatSplit
		}
		model.SelectionStart = 0
		model.SelectionEnd = 0
		model.SelectionSide = ""
		updateView(model)
		return model, nil
	})

	h.HandleEvent("init-diff-format", func(ctx context.Context, s *live.Socket, p live.Params) (any, error) {
		model := getModel(s, rs.Model)
		format := p.String("format")
		if format == string(DiffFormatUnified) || format == string(DiffFormatSplit) {
			model.DiffFormat = DiffFormat(format)
			updateView(model)
		}
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

func deleteComment(model *ReviewModel, id int) {
	for i := range model.Comments {
		if model.Comments[i].ID == id {
			model.Comments = append(model.Comments[:i], model.Comments[i+1:]...)
			if model.EditingCommentID == id {
				model.EditingCommentID = 0
			}
			return
		}
	}
}

func editComment(model *ReviewModel, id int, text string) error {
	if text == "" {
		return fmt.Errorf("comment text is required")
	}
	for i := range model.Comments {
		if model.Comments[i].ID == id {
			model.Comments[i].Text = text
			model.EditingCommentID = 0
			return nil
		}
	}
	return fmt.Errorf("comment not found")
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

func localFileHandler(root string) http.Handler {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		rootAbs = root
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimSpace(r.URL.Query().Get("path"))
		if rel == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}
		rel = filepath.Clean(rel)
		if filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		target := filepath.Join(rootAbs, rel)
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		relToRoot, err := filepath.Rel(rootAbs, targetAbs)
		if err != nil || strings.HasPrefix(relToRoot, "..") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		http.ServeFile(w, r, targetAbs)
	})
}
