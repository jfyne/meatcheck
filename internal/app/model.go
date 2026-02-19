package app

import (
	"html/template"
	"sync"
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
	Files                []File
	DiffFiles            []DiffFile
	Tree                 []TreeItem
	SelectedPath         string
	SelectedLabel        string
	CodeViewKey          string
	Mode                 ViewMode
	RenderFile           bool
	RenderComments       bool
	Prompt               string
	PromptHTML           template.HTML
	SelectionStart       int
	SelectionEnd         int
	CommentDraft         string
	Comments             []Comment
	Ranges               map[string][]LineRange
	MarkdownRenderByPath map[string]bool
	ViewFile             ViewFile
	ViewDiff             ViewDiffFile
	Error                string
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
