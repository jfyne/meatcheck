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
		Files:                files,
		DiffFiles:            diffFiles,
		SelectedPath:         "",
		SelectedLabel:        "",
		Mode:                 mode,
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
