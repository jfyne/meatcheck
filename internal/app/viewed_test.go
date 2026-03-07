package app

import "testing"

// TestNextUnviewedFileUngroupedFileMode verifies that when in ungrouped file
// mode with the first file currently selected and already viewed, the function
// returns the path of the second file.
//
// Scenario: Next unviewed file found (file mode) — current is first of 3, first is viewed, returns second
func TestNextUnviewedFileUngroupedFileMode(t *testing.T) {
	model := &ReviewModel{
		Mode: ModeFile,
		Files: []File{
			{Path: "a.go"},
			{Path: "b.go"},
			{Path: "c.go"},
		},
		SelectedPath: "a.go",
		Viewed: map[string]bool{
			"a.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "b.go" {
		t.Errorf("nextUnviewedFile: got %q, want %q", got, "b.go")
	}
}

// TestNextUnviewedFileUngroupedWrapAround verifies that when the current file
// is the last in the list and only the first file is unviewed, the search
// wraps around and returns the first file's path.
//
// Scenario: Wrap-around — current is last, next unviewed is first
func TestNextUnviewedFileUngroupedWrapAround(t *testing.T) {
	model := &ReviewModel{
		Mode: ModeFile,
		Files: []File{
			{Path: "a.go"},
			{Path: "b.go"},
			{Path: "c.go"},
		},
		SelectedPath: "c.go",
		Viewed: map[string]bool{
			"b.go": true,
			"c.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "a.go" {
		t.Errorf("nextUnviewedFile: got %q, want %q", got, "a.go")
	}
}

// TestNextUnviewedFileUngroupedAllViewed verifies that when every file in the
// list has been viewed, the function returns an empty string.
//
// Scenario: All viewed returns "" — all files viewed, returns empty string
func TestNextUnviewedFileUngroupedAllViewed(t *testing.T) {
	model := &ReviewModel{
		Mode: ModeFile,
		Files: []File{
			{Path: "a.go"},
			{Path: "b.go"},
			{Path: "c.go"},
		},
		SelectedPath: "a.go",
		Viewed: map[string]bool{
			"a.go": true,
			"b.go": true,
			"c.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "" {
		t.Errorf("nextUnviewedFile: got %q, want empty string", got)
	}
}

// TestNextUnviewedFileDiffMode verifies that in diff mode the function uses
// DiffFiles rather than Files to build the ordered path list.
//
// Scenario: Diff mode — same as file mode but using DiffFiles
func TestNextUnviewedFileDiffMode(t *testing.T) {
	model := &ReviewModel{
		Mode: ModeDiff,
		DiffFiles: []DiffFile{
			{Path: "x.go"},
			{Path: "y.go"},
			{Path: "z.go"},
		},
		SelectedPath: "x.go",
		Viewed: map[string]bool{
			"x.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "y.go" {
		t.Errorf("nextUnviewedFile: got %q, want %q", got, "y.go")
	}
}

// TestNextUnviewedFileGroupedWithinGroup verifies that when HasGroups is true
// and the next unviewed file is in the same group as the current file, that
// path is returned.
//
// Scenario: Within-group advance — next unviewed in same group
func TestNextUnviewedFileGroupedWithinGroup(t *testing.T) {
	model := &ReviewModel{
		HasGroups: true,
		Groups: []Group{
			{Name: "backend", Files: []string{"a.go", "b.go", "c.go"}},
			{Name: "frontend", Files: []string{"d.js", "e.js"}},
		},
		SelectedPath: "a.go",
		Viewed: map[string]bool{
			"a.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "b.go" {
		t.Errorf("nextUnviewedFile: got %q, want %q", got, "b.go")
	}
}

// TestNextUnviewedFileGroupedCrossGroup verifies that when the last file of
// group A is viewed, the search advances into group B and returns the first
// unviewed file there.
//
// Scenario: Cross-group advance — last file in group A viewed, advances to first unviewed in group B
func TestNextUnviewedFileGroupedCrossGroup(t *testing.T) {
	model := &ReviewModel{
		HasGroups: true,
		Groups: []Group{
			{Name: "backend", Files: []string{"a.go", "b.go"}},
			{Name: "frontend", Files: []string{"c.js", "d.js"}},
		},
		SelectedPath: "b.go",
		Viewed: map[string]bool{
			"a.go": true,
			"b.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "c.js" {
		t.Errorf("nextUnviewedFile: got %q, want %q", got, "c.js")
	}
}

// TestNextUnviewedFileGroupedAllViewed verifies that when all files across all
// groups are viewed, the function returns an empty string.
//
// Scenario: All viewed in grouped mode returns ""
func TestNextUnviewedFileGroupedAllViewed(t *testing.T) {
	model := &ReviewModel{
		HasGroups: true,
		Groups: []Group{
			{Name: "backend", Files: []string{"a.go", "b.go"}},
			{Name: "frontend", Files: []string{"c.js"}},
		},
		SelectedPath: "a.go",
		Viewed: map[string]bool{
			"a.go": true,
			"b.go": true,
			"c.js": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "" {
		t.Errorf("nextUnviewedFile: got %q, want empty string", got)
	}
}

// TestNextUnviewedFileNilViewedMap verifies that a nil Viewed map is treated as
// every file being unviewed, so the next file after the current selection is
// returned.
//
// Scenario: nil viewed map — treated as all unviewed, returns next file
func TestNextUnviewedFileNilViewedMap(t *testing.T) {
	model := &ReviewModel{
		Mode: ModeFile,
		Files: []File{
			{Path: "a.go"},
			{Path: "b.go"},
			{Path: "c.go"},
		},
		SelectedPath: "a.go",
		Viewed:       nil,
	}

	got := nextUnviewedFile(model)
	if got != "b.go" {
		t.Errorf("nextUnviewedFile: got %q, want %q", got, "b.go")
	}
}

// TestNextUnviewedFileSelectedPathNotFound verifies that when SelectedPath is
// not present in the ordered file list, the function returns the first unviewed
// file (or "" if all are viewed).
//
// Scenario: Selected path not found — returns first unviewed file or ""
func TestNextUnviewedFileSelectedPathNotFound(t *testing.T) {
	model := &ReviewModel{
		Mode: ModeFile,
		Files: []File{
			{Path: "a.go"},
			{Path: "b.go"},
			{Path: "c.go"},
		},
		SelectedPath: "nonexistent.go",
		Viewed: map[string]bool{
			"a.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "b.go" {
		t.Errorf("nextUnviewedFile: got %q, want %q", got, "b.go")
	}
}

// TestNextUnviewedFileGroupedIncludesOtherFiles verifies that when HasGroups
// is true, files not assigned to any group (the "Other" group) are still
// included in the navigation order after all grouped files.
func TestNextUnviewedFileGroupedIncludesOtherFiles(t *testing.T) {
	model := &ReviewModel{
		Mode:      ModeFile,
		HasGroups: true,
		Groups: []Group{
			{Name: "backend", Files: []string{"a.go"}},
		},
		Files: []File{
			{Path: "a.go"},
			{Path: "ungrouped.go"},
		},
		SelectedPath: "a.go",
		Viewed: map[string]bool{
			"a.go": true,
		},
	}

	got := nextUnviewedFile(model)
	if got != "ungrouped.go" {
		t.Errorf("nextUnviewedFile: got %q, want %q (ungrouped file should be reachable)", got, "ungrouped.go")
	}
}
