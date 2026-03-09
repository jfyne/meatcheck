package app

import (
	"html/template"
	"sync"
)

type Comment struct {
	ID        int    `json:"id"`
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Side      string `json:"side,omitempty"`
	Text      string `json:"text"`
}

type Group struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

type File struct {
	Path      string
	PathSlash string
	Lines     []string
}

type TreeItem struct {
	Name        string
	Path        string
	Depth       int
	IsDir       bool
	Selected    bool
	IsGroup     bool
	Viewed      bool
	HasComments bool
	GroupName   string
	GroupActive bool
}

type ViewLine struct {
	Number    int
	Text      string
	HTML      template.HTML
	Selected  bool
	Commented bool
	Comments  []ViewComment
}

type MarkdownBlock struct {
	StartLine int
	EndLine   int
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
	MarkdownBlocks   []MarkdownBlock
}

type ViewMode string

const (
	ModeFile ViewMode = "file"
	ModeDiff ViewMode = "diff"
)

type DiffFormat string

const (
	DiffFormatUnified DiffFormat = "unified"
	DiffFormatSplit   DiffFormat = "split"
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

type ViewDiffSide struct {
	Line      int
	Kind      DiffLineKind
	Text      string
	HTML      template.HTML
	Empty     bool
	Selected  bool
	Commented bool
	Comments  []ViewComment
}

type ViewDiffRow struct {
	Left  ViewDiffSide
	Right ViewDiffSide
}

type ViewDiffSplitHunk struct {
	Header string
	Rows   []ViewDiffRow
}

type ViewComment struct {
	Comment
	Rendered template.HTML
	Editing  bool
}

type LineRange struct {
	Start int
	End   int
}

type ReviewModel struct {
	Files                []File
	DiffFiles            []DiffFile
	Viewed               map[string]bool
	Groups               []Group
	HasGroups            bool
	Tree                 []TreeItem
	SelectedPath         string
	SelectedLabel        string
	CodeViewKey          string
	Mode                 ViewMode
	RenderFile           bool
	RenderComments       bool
	SidebarCollapsed     bool
	Prompt               string
	PromptHTML           template.HTML
	SelectionStart       int
	SelectionEnd         int
	SelectionSide        string
	CommentDraft         string
	Comments             []Comment
	NextCommentID        int
	EditingCommentID     int
	Ranges               map[string][]LineRange
	MarkdownRenderByPath map[string]bool
	ViewFile             ViewFile
	ViewDiff             ViewDiffFile
	ViewDiffSplit        []ViewDiffSplitHunk
	DiffFormat           DiffFormat
	Error                string
}

// diffOldLineExists reports whether the given old-side line number exists in
// the diff hunks for the specified file path. Add lines are excluded because
// they have no old-side representation.
func diffOldLineExists(files []DiffFile, path string, oldLine int) bool {
	file := findDiffFile(files, path)
	if file == nil {
		return false
	}
	for _, h := range file.Hunks {
		for _, dl := range h.Lines {
			if dl.OldLine == oldLine && dl.Kind != DiffAdd {
				return true
			}
		}
	}
	return false
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
	Groups  []Group
}
