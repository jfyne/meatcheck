package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type DiffLineKind string

const (
	DiffContext DiffLineKind = "context"
	DiffAdd     DiffLineKind = "add"
	DiffDel     DiffLineKind = "del"
)

type DiffLine struct {
	Kind    DiffLineKind
	OldLine int
	NewLine int
	Text    string
}

type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

type DiffFile struct {
	OldPath string
	NewPath string
	Path    string
	Hunks   []DiffHunk
}

var hunkHeaderRE = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func parseUnifiedDiff(input string) ([]DiffFile, error) {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	var files []DiffFile
	var curFile *DiffFile
	var curHunk *DiffHunk
	var oldLine, newLine int

	flushFile := func() {
		if curFile == nil {
			return
		}
		files = append(files, *curFile)
		curFile = nil
		curHunk = nil
	}

	for _, raw := range lines {
		if strings.HasPrefix(raw, "diff --git ") {
			flushFile()
			curFile = &DiffFile{}
			continue
		}
		if strings.HasPrefix(raw, "--- ") {
			path := strings.TrimSpace(strings.TrimPrefix(raw, "--- "))
			if curFile == nil {
				curFile = &DiffFile{}
			}
			curFile.OldPath = normalizeDiffPath(path)
			continue
		}
		if strings.HasPrefix(raw, "+++ ") {
			path := strings.TrimSpace(strings.TrimPrefix(raw, "+++ "))
			if curFile == nil {
				curFile = &DiffFile{}
			}
			curFile.NewPath = normalizeDiffPath(path)
			curFile.Path = pickDiffPath(curFile.OldPath, curFile.NewPath)
			continue
		}
		if strings.HasPrefix(raw, "@@ ") {
			if curFile == nil {
				curFile = &DiffFile{}
			}
			m := hunkHeaderRE.FindStringSubmatch(raw)
			if m == nil {
				return nil, fmt.Errorf("invalid hunk header: %s", raw)
			}
			oldStart := mustAtoi(m[1])
			oldCount := parseOptionalCount(m[2])
			newStart := mustAtoi(m[3])
			newCount := parseOptionalCount(m[4])
			h := DiffHunk{OldStart: oldStart, OldCount: oldCount, NewStart: newStart, NewCount: newCount}
			curFile.Hunks = append(curFile.Hunks, h)
			curHunk = &curFile.Hunks[len(curFile.Hunks)-1]
			oldLine = oldStart
			newLine = newStart
			continue
		}
		if curHunk == nil {
			continue
		}
		if strings.HasPrefix(raw, "\\ No newline at end of file") {
			continue
		}
		if raw == "" {
			// Empty context line in a diff is represented as a single space; if not,
			// treat as context without prefix.
			curHunk.Lines = append(curHunk.Lines, DiffLine{Kind: DiffContext, OldLine: oldLine, NewLine: newLine, Text: ""})
			oldLine++
			newLine++
			continue
		}
		prefix := raw[0]
		text := raw[1:]
		switch prefix {
		case ' ':
			curHunk.Lines = append(curHunk.Lines, DiffLine{Kind: DiffContext, OldLine: oldLine, NewLine: newLine, Text: text})
			oldLine++
			newLine++
		case '+':
			curHunk.Lines = append(curHunk.Lines, DiffLine{Kind: DiffAdd, OldLine: 0, NewLine: newLine, Text: text})
			newLine++
		case '-':
			curHunk.Lines = append(curHunk.Lines, DiffLine{Kind: DiffDel, OldLine: oldLine, NewLine: 0, Text: text})
			oldLine++
		default:
			// treat as context if malformed
			curHunk.Lines = append(curHunk.Lines, DiffLine{Kind: DiffContext, OldLine: oldLine, NewLine: newLine, Text: raw})
			oldLine++
			newLine++
		}
	}

	flushFile()
	return files, nil
}

func normalizeDiffPath(path string) string {
	if path == "/dev/null" {
		return ""
	}
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		path = path[2:]
	}
	path = strings.TrimPrefix(path, "./")
	return filepath.ToSlash(path)
}

func pickDiffPath(oldPath, newPath string) string {
	if newPath != "" {
		return newPath
	}
	return oldPath
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	if n == 0 {
		return 0
	}
	return n
}

func parseOptionalCount(s string) int {
	if s == "" {
		return 1
	}
	return mustAtoi(s)
}
